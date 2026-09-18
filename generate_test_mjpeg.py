import io
from PIL import Image, ImageDraw

def generate_mjpeg(filename, width=320, height=480, frames=40):
    with open(filename, 'wb') as f:
        for i in range(frames):
            # Create a black background
            img = Image.new('RGB', (width, height), color=(20, 20, 20))
            draw = ImageDraw.Draw(img)
            
            # Animate a bouncing circle
            y_pos = int(height/2 + 100 * __import__('math').sin(i * 0.3))
            
            draw.ellipse(
                (width/2 - 30, y_pos - 30, width/2 + 30, y_pos + 30),
                fill=(255, 100, 100)
            )
            
            # Draw a frame number
            draw.text((10, 10), f"Frame {i+1}/{frames}", fill=(255, 255, 255))
            
            # Save to JPEG buffer
            buf = io.BytesIO()
            img.save(buf, format='JPEG', quality=85)
            
            # Write to file
            f.write(buf.getvalue())
            
    print(f"Generado {filename} con éxito.")

if __name__ == '__main__':
    generate_mjpeg('test.mjpeg')
