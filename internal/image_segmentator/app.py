from fastapi import FastAPI, File, UploadFile
from PIL import Image
import io
import uuid

from fastapi.responses import JSONResponse
from process import load_seg_model, generate_mask, get_palette
import base64
import asyncio
import uvicorn
from concurrent.futures import ProcessPoolExecutor
import multiprocessing

app = FastAPI()

# Load model at startup once
model = load_seg_model(device='cpu')
palette = get_palette(4)

process_pool = ProcessPoolExecutor(max_workers=max(1, multiprocessing.cpu_count() - 1))

async def process_image(img, req_id):
    """Process a single image in a separate process to avoid GIL limitations"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(
        process_pool,
        generate_mask,
        req_id, img, model, palette, 'cpu'
    )

@app.post("/segment/")
async def segment_clothes(file: UploadFile = File(...)):
    try:
        img_bytes = await file.read()
        img = Image.open(io.BytesIO(img_bytes)).convert('RGB')

        req_id = str(uuid.uuid4())
        encoded_parts = await process_image(img, req_id)
        if not encoded_parts:
            raise ValueError("Segmentation returned no results")

        result = []
        for filename, image_data, metadata in encoded_parts:
            encoded_image = base64.b64encode(image_data).decode('utf-8')
            result.append({
                "filename": filename,
                "image": encoded_image,  # send image as hex string or base64
                "metadata": metadata
            })

        return {"segmented_images": result}
    except Exception as e:
        print(f"Error during segmentation: {e}")
        return JSONResponse(status_code=500, content={"error": str(e)})

if __name__ == "__main__":
    # Run with multiple workers
    uvicorn.run("app:app", host="0.0.0.0", port=8000, workers=4)
