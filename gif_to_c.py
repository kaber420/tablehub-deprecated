import os

def gif_to_c_array(input_path, output_path, array_name):
    with open(input_path, "rb") as f:
        data = f.read()
    
    with open(output_path, "w") as f:
        f.write("#include \"lvgl.h\"\n")
        f.write(f"const uint8_t {array_name}[] = {{\n")
        
        for i, byte in enumerate(data):
            f.write(f"0x{byte:02X}, ")
            if (i + 1) % 16 == 0:
                f.write("\n")
                
        f.write("\n};\n")
        f.write(f"const size_t {array_name}_size = {len(data)};\n")

if __name__ == "__main__":
    gif_to_c_array("assets/small_test.gif", "firmware/src/UI/Views/small_test_gif.c", "small_test_gif")
