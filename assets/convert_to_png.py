from PIL import Image
import sys

def convert(input_path, output_path):
    img = Image.open(input_path).convert("RGBA")
    # Resize image to a reasonable size if it's too big, LVGL on ESP32 has limited RAM
    # Maybe 100x100?
    img.thumbnail((120, 120))
    img.save(output_path, "PNG")

if __name__ == "__main__":
    convert(sys.argv[1], sys.argv[2])
