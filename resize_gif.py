import os
from PIL import Image

input_path = "assets/video-from-rawpixel-id-26466765-gif.gif"
output_path = "assets/small_test.gif"

def resize_gif(input_path, output_path, target_width=135, target_height=240, frame_skip=2):
    print(f"Opening {input_path}...")
    img = Image.open(input_path)
    
    frames = []
    
    # Iterate over frames
    for i in range(0, getattr(img, "n_frames", 1), frame_skip):
        img.seek(i)
        # Convert to RGBA to ensure transparency is handled well if needed, then RGB
        frame = img.copy()
        frame = frame.convert("RGB") # RGB is better for simple gifs without transparency
        
        # Resize frame
        resized = frame.resize((target_width, target_height), Image.Resampling.LANCZOS)
        frames.append(resized)
    
    print(f"Saving to {output_path} with {len(frames)} frames...")
    # Save frames as new GIF
    frames[0].save(
        output_path,
        save_all=True,
        append_images=frames[1:],
        loop=0,
        optimize=True
    )
    print("Done! Original size:", os.path.getsize(input_path)/1024/1024, "MB")
    print("New size:", os.path.getsize(output_path)/1024/1024, "MB")

if __name__ == "__main__":
    resize_gif(input_path, output_path)
