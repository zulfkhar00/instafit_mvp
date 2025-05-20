from network import U2NET
from metadata_retriever import get_metadata

import os
from PIL import Image
import cv2
import gdown
import argparse
import numpy as np
import sys
import io # Import io for in-memory binary streams
import zipfile # Import zipfile
import uuid
import json
import time

import torch
import torch.nn.functional as F
import torchvision.transforms as transforms

from collections import OrderedDict
# from options import opt


def load_checkpoint(model):
    current_dir = os.path.dirname(os.path.abspath(__file__))
    # onnx_model_path = os.path.join(current_dir, "fashion-clip", "onnx", "model.onnx")
    checkpoint_path = os.path.join(current_dir, "model", "cloth_segm.pth")
    # checkpoint_path = "model/cloth_segm.pth"
    if not os.path.exists(checkpoint_path):
        print("----No checkpoints at given path----")
        return
    model_state_dict = torch.load(checkpoint_path, map_location=torch.device("cpu"))
    new_state_dict = OrderedDict()
    for k, v in model_state_dict.items():
        name = k[7:]  # remove `module.`
        new_state_dict[name] = v

    model.load_state_dict(new_state_dict)
    print("----checkpoints loaded from path: {}----".format(checkpoint_path))
    return model


def get_palette(num_cls):
    """ Returns the color map for visualizing the segmentation mask.
    Args:
        num_cls: Number of classes
    Returns:
        The color map
    """
    n = num_cls
    palette = [0] * (n * 3)
    for j in range(0, n):
        lab = j
        palette[j * 3 + 0] = 0
        palette[j * 3 + 1] = 0
        palette[j * 3 + 2] = 0
        i = 0
        while lab:
            palette[j * 3 + 0] |= (((lab >> 0) & 1) << (7 - i))
            palette[j * 3 + 1] |= (((lab >> 1) & 1) << (7 - i))
            palette[j * 3 + 2] |= (((lab >> 2) & 1) << (7 - i))
            i += 1
            lab >>= 3
    return palette


class Normalize_image(object):
    """Normalize given tensor into given mean and standard dev

    Args:
        mean (float): Desired mean to substract from tensors
        std (float): Desired std to divide from tensors
    """

    def __init__(self, mean, std):
        assert isinstance(mean, (float))
        if isinstance(mean, float):
            self.mean = mean

        if isinstance(std, float):
            self.std = std

        self.normalize_1 = transforms.Normalize(self.mean, self.std)
        self.normalize_3 = transforms.Normalize([self.mean] * 3, [self.std] * 3)
        self.normalize_18 = transforms.Normalize([self.mean] * 18, [self.std] * 18)

    def __call__(self, image_tensor):
        if image_tensor.shape[0] == 1:
            return self.normalize_1(image_tensor)

        elif image_tensor.shape[0] == 3:
            return self.normalize_3(image_tensor)

        elif image_tensor.shape[0] == 18:
            return self.normalize_18(image_tensor)

        else:
            assert "Please set proper channels! Normlization implemented only for 1, 3 and 18"




def apply_transform(img):
    transforms_list = []
    transforms_list += [transforms.ToTensor()]
    transforms_list += [Normalize_image(0.5, 0.5)]
    transform_rgb = transforms.Compose(transforms_list)
    return transform_rgb(img)



def generate_mask(req_id, input_image, net, palette, device = 'cpu'):
    #img = Image.open(input_image).convert('RGB')
    img = input_image
    img_size = img.size
    img = img.resize((768, 768), Image.BICUBIC)
    image_tensor = apply_transform(img)
    image_tensor = torch.unsqueeze(image_tensor, 0)

    with torch.no_grad():
        output_tensor = net(image_tensor.to(device))
        output_tensor = F.log_softmax(output_tensor[0], dim=1)
        output_tensor = torch.max(output_tensor, dim=1, keepdim=True)[1]
        output_tensor = torch.squeeze(output_tensor, dim=0)
        output_arr = output_tensor.cpu().numpy()

    classes_to_save = []

    # Check which classes are present in the image
    for cls in range(1, 4):  # Exclude background class (0)
        if np.any(output_arr == cls):
            classes_to_save.append(cls)

    # Save alpha masks
    img_np = np.array(input_image)
    encoded_images = [] # List to hold (filename, image_bytes)

    for cls in classes_to_save:
        alpha_mask = (output_arr == cls).astype(np.uint8) * 255
        alpha_mask = alpha_mask[0]  # make it 2D
        alpha_mask_img = Image.fromarray(alpha_mask, mode='L').resize(img_size, Image.BICUBIC)

        alpha_mask_np = np.array(alpha_mask_img)
        masked_img_np = cv2.bitwise_and(img_np, img_np, mask=alpha_mask_np)

        masked_img = Image.fromarray(masked_img_np)
        metadata = get_metadata(masked_img)

        img_byte_arr = io.BytesIO()
        masked_img.save(img_byte_arr, format='JPEG', quality=90)
        image_bytes = img_byte_arr.getvalue()
        encoded_images.append((f'clothing_{cls}_{req_id}.png', image_bytes, metadata))

    return encoded_images



def check_or_download_model(file_path):
    if not os.path.exists(file_path):
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        url = "https://drive.google.com/uc?id=11xTBALOeUkyuaK3l60CpkYHLTmv7k3dY"
        gdown.download(url, file_path, quiet=False)
        print("Model downloaded successfully.")
    else:
        print("Model already exists.")


def load_seg_model(device='cpu'):
    net = U2NET(in_ch=3, out_ch=4)
    # check_or_download_model(checkpoint_path)
    net = load_checkpoint(net)
    net = net.to(device)
    net = net.eval()

    return net


def main(args):
    start = time.time()
    device = 'cuda' if args.cuda else 'cpu'

    # Create an instance of your model
    model = load_seg_model(device=device)

    palette = get_palette(4)

    # Read image bytes directly from stdin
    image_bytes = sys.stdin.buffer.read()
    try:
        img = Image.open(io.BytesIO(image_bytes)).convert('RGB')
    except Exception as e:
        sys.stderr.write(f"Error reading image from stdin: {e}\n")
        sys.exit(1)

    unique_id = str(uuid.uuid4())
    encoded_clothing_parts = generate_mask(unique_id, img, net=model, palette=palette, device=device)

    if not encoded_clothing_parts:
        sys.stderr.write("Warning: No clothing parts were segmented.\n")

    # --- Create an in-memory ZIP archive ---
    zip_buffer = io.BytesIO()
    try:
        with zipfile.ZipFile(zip_buffer, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for filename, img_bytes, metadata in encoded_clothing_parts:
                zipf.writestr(filename, img_bytes)
                metadata_filename = filename[:-4] + ".json"
                metadata_json_string = json.dumps(metadata, indent=None)
                metadata_bytes = metadata_json_string.encode('utf-8')
                zipf.writestr(metadata_filename, metadata_bytes)
    except Exception as e:
         sys.stderr.write(f"Error creating zip archive: {e}\n")
         sys.exit(1)
    # --- Write the ZIP archive bytes to stdout ---
    try:
        sys.stdout.buffer.write(zip_buffer.getvalue())
        sys.stdout.buffer.flush()
        sys.stderr.write("Zip archive written to stdout.\n") # Log success to stderr
        execution_time = time.time() - start
        sys.stderr.write(f"[process] Execution time: {execution_time:.4f} seconds\n")
        sys.exit(0) # Exit successfully
    except Exception as e:
        sys.stderr.write(f"Error writing zip archive to stdout: {e}\n")
        sys.exit(1) # Exit with error code


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Help to set arguments for Cloth Segmentation.')
    # parser.add_argument('--image', type=str, help='Path to the input image')
    parser.add_argument('--cuda', action='store_true', help='Enable CUDA (default: False)')
    args = parser.parse_args()

    main(args)
