import os
from PIL import Image
import subprocess

LVGL_SCRIPT = "/home/kaber420/Documentos/proyectos/tablehub2/firmware/.pio/libdeps/esp32/lvgl/scripts/LVGLImage.py"

TARGET_W = 140
TARGET_H = 120

for file in os.listdir("."):
    if file.endswith(".png"):
        hash_name = file.replace(".png", "")
        img = Image.open(file).convert("RGBA")
        
        # Crop to fill logic
        img_ratio = img.width / img.height
        target_ratio = TARGET_W / TARGET_H
        
        if img_ratio > target_ratio:
            # Image is wider, crop width
            new_w = int(img.height * target_ratio)
            offset = (img.width - new_w) // 2
            img = img.crop((offset, 0, offset + new_w, img.height))
        else:
            # Image is taller, crop height
            new_h = int(img.width / target_ratio)
            offset = (img.height - new_h) // 2
            img = img.crop((0, offset, img.width, offset + new_h))
            
        img = img.resize((TARGET_W, TARGET_H), Image.Resampling.LANCZOS)
        
        temp_png = f"temp_{hash_name}_{TARGET_W}x{TARGET_H}.png"
        img.save(temp_png, "PNG")
        
        # Convert to BIN
        out_name = f"{hash_name}_{TARGET_W}x{TARGET_H}"
        cmd = ["/home/kaber420/Documentos/proyectos/tablehub2/firmware/venv/bin/python3", LVGL_SCRIPT, "--ofmt", "BIN", "--cf", "RGB565", temp_png, "--name", out_name, "-o", "."]
        subprocess.run(cmd)
        
        os.remove(temp_png)
        print(f"Generated {out_name}.bin")
