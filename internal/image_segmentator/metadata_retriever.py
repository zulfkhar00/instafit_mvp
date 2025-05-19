import torch
import clip
from PIL import Image
import numpy as np
from sklearn.cluster import KMeans
from colormath.color_objects import sRGBColor, LabColor
from colormath.color_conversions import convert_color
from matplotlib import colors as mcolors
import time

start_time = time.time()

device = "cuda" if torch.cuda.is_available() else "cpu"
model, preprocess = clip.load("ViT-B/32", device=device)
COLOR_NAMES = dict(mcolors.CSS4_COLORS, **mcolors.XKCD_COLORS)

def classify(img: Image.Image, labels, threshold=0.2):
    image = preprocess(img).unsqueeze(0).to(device)
    text = clip.tokenize(labels).to(device)

    with torch.no_grad():
        logits, _ = model(image, text)
        probs = logits.softmax(dim=-1).cpu().numpy()[0]

    return [label for label, prob in zip(labels, probs) if prob >= threshold]
    # return {label: float(prob) for label, prob in zip(labels, probs) if prob >= threshold}

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
    # 1. Season classification
    seasons = ["spring", "summer", "fall", "winter"]
    season = classify(img, seasons)

    # 2. Occasion classification
    occasions = ["daily", "work", "date", "formal", "travel", "home", "party", "sport", "special", "school", "beach"]
    occasion = classify(img, occasions)

    # 3. Clothing category
    categories = ["tops", "bottoms"]
    category = classify(img, categories)

    # 4. Clothing type
    types = ["t-shirt", "long-sleeve t-shirt", "sleeveless t-shirt", "polo shirt", "tanks & camis", "crop tops", "blouses", "shirts", "sweatshirts", "hoodies", "sweaters", "sweater vests", "cardigan tops", "sports tops", "bodysuits"]
    clothing_type = classify(img, types)

    # 5. Clothing colors
    colors = detect_colors(img)

    # 6. Clothing style
    styles = ["casual", "comfortable", "business casual", "formal", "modern", "classic", "minimalist", "bohemian", "luxury", "sporty", "athleisure", "affordable", "trendy", "premium", "kidcore", "basic", "artistic", "dress-up", "hipster", "feminine", "chic", "street"]
    style = classify(img, styles)

    # 7. Clothing fit
    fits = ["slim", "regular", "loose", "oversized"]
    fit = classify(img, fits)

    # 8. Neckline classification
    necklines = ["round neckline", "scoop neckline", "boat neckline", "v-neck neckline", "deep-v neckline", "square neckline", "surplice neckline", "shirts-collar neckline", "standup-collar neckline", "wide-collar neckline", "mockneck neckline", "turtleneck neckline", "strapless neckline", "thick-strap neckline", "thin-strap neckline", "sweetheart neckline", "off-the-shoulder neckline", "asymmetric neckline", "halter neckline", "illusion neckline", "keyhole neckline", "suit-collar neckline"]
    neckline = classify(img, necklines)

    # 9. Sleeve length classification
    sleeves = ["sleeveless", "cap sleeve", "short sleeve", "3/4 sleeve", "long sleeve"]
    sleeve = classify(img, sleeves)

    # 10. Clothing length
    lengths = ["crop length", "waist length", "hip length", "knee length"]
    clothing_length = classify(img, lengths)

    end_time = time.time()
    execution_time = end_time - start_time
    print(f"Execution time: {execution_time:.4f} seconds")

    return {
        "seasons": season,
        "occasions": occasion,
        "categories": category,
        "types": clothing_type,
        "colors": colors,
        "styles": style,
        "fits": fit,
        "necklines": neckline,
        "sleeves": sleeve,
        "lengths": clothing_length,
    }
