import onnxruntime as ort
from transformers import CLIPProcessor
from PIL import Image
import numpy as np
from sklearn.cluster import KMeans
from colormath.color_objects import sRGBColor, LabColor
from colormath.color_conversions import convert_color
from matplotlib import colors as mcolors

onnx_model_path = "/Users/zmaukey/Desktop/instafit_mvp/internal/image_segmentator/fashion-clip/onnx/model.onnx"
processor_path = "/Users/zmaukey/Desktop/instafit_mvp/internal/image_segmentator/fashion-clip"
COLOR_NAMES = dict(mcolors.CSS4_COLORS, **mcolors.XKCD_COLORS)

session = ort.InferenceSession(onnx_model_path)
processor = CLIPProcessor.from_pretrained(processor_path, use_fast=False)

def classify(img: Image.Image, labels, threshold=0.5):
    inputs = processor(text=labels, images=img, return_tensors="np", padding=True)

    onnx_inputs = {
        "input_ids": inputs["input_ids"],
        "attention_mask": inputs["attention_mask"],
        "pixel_values": inputs["pixel_values"]
    }

    logits = session.run(None, onnx_inputs)[0]
    probs = np.exp(logits) / np.sum(np.exp(logits), axis=1, keepdims=True)
    probs = probs[0]

    return [label for label, prob in zip(labels, probs) if prob >= threshold]

def closest_color_lab(hex_color):
    requested_rgb = np.array(mcolors.to_rgb(hex_color)) * 255
    requested_lab = convert_color(sRGBColor(*requested_rgb), LabColor)

    min_distance = float('inf')
    closest_name = None

    for name, hex_std in COLOR_NAMES.items():
        std_rgb = np.array(mcolors.to_rgb(hex_std)) * 255
        std_lab = convert_color(sRGBColor(*std_rgb), LabColor)

        distance = ((requested_lab.lab_l - std_lab.lab_l)**2 +
                    (requested_lab.lab_a - std_lab.lab_a)**2 +
                    (requested_lab.lab_b - std_lab.lab_b)**2)**0.5

        if distance < min_distance:
            min_distance = distance
            closest_name = name.replace('xkcd:', '').replace('css4:', '')

    return closest_name


def detect_colors(img: Image.Image, num_colors=3, alpha_threshold=200):
    image = img.convert('RGBA')
    pixels = np.array(image)

    alpha = pixels[:, :, 3]
    mask = alpha > alpha_threshold
    rgb_pixels = pixels[:, :, :3][mask]

    if len(rgb_pixels) == 0:
        raise ValueError("No valid pixels found for color detection.")

    kmeans = KMeans(n_clusters=num_colors, random_state=42).fit(rgb_pixels)
    dominant_colors = np.round(kmeans.cluster_centers_).astype(int)

    res = []
    for color in dominant_colors:
        r, g, b = color
        hex_color = f'#{r:02x}{g:02x}{b:02x}'
        res.append(closest_color_lab(hex_color))
    return res


def get_metadata(img: Image.Image) -> dict[str, list[str]]:
    season = classify(img, ["spring", "summer", "fall", "winter"])
    occasion = classify(img, ["daily", "work", "date", "formal", "travel", "home", "party", "sport", "special", "school", "beach"])
    category = classify(img, ["tops", "bottoms"])

    tops_types = ["t-shirt", "long-sleeve t-shirt", "sleeveless t-shirt", "polo shirt", "tanks & camis", "crop tops", "blouses", "shirts", "sweatshirts", "hoodies", "sweaters", "sweater vests", "cardigan tops", "sports tops", "bodysuits"]
    bottoms_types = ["jeans", "trousers", "pants", "shorts", "skirts", "leggings", "joggers", "sweatpants", "chinos", "cargo pants", "culottes", "capris", "maxi skirt", "mini skirt", "midi skirt", "athletic shorts", "denim shorts", "formal pants", "track pants", "bike shorts"]

    clothing_type = classify(img, tops_types if "tops" in category else bottoms_types)
    colors = detect_colors(img)
    style = classify(img, ["casual", "comfortable", "business casual", "formal", "modern", "classic", "minimalist", "bohemian", "luxury", "sporty", "athleisure", "affordable", "trendy", "premium", "kidcore", "basic", "artistic", "dress-up", "hipster", "feminine", "chic", "street"])
    fit = classify(img, ["slim", "regular", "loose", "oversized"])

    res = {"seasons": season, "occasions": occasion, "categories": category, "types": clothing_type, "colors": colors, "styles": style, "fits": fit}

    if "tops" in category:
        res["necklines"] = classify(img, ["round neckline", "v-neck neckline", "turtleneck neckline"])
        res["sleeves"] = classify(img, ["sleeveless", "short sleeve", "long sleeve"])
        res["clothing_lengths"] = classify(img, ["short", "regular", "long"])
    else:
        res["waist_styles"] = classify(img, ["high-waisted", "mid-rise", "low-rise"])
        res["clothing_lengths"] = classify(img, ["short", "knee-length", "midi", "ankle-length", "full-length"])

    return res
