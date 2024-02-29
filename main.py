import os
import re

def replace_image_path(root_dir):
    """
    Recursively search for .md files in root_dir and subfolders,
    remove the 'static/' part from the image paths, and write the changes back.
    
    :param root_dir: The root directory to search for .md files.
    """
    img_path_regex = re.compile(r'!\[(.*?)\]\((static/)(.*?)\)')

    for dirpath, _, filenames in os.walk(root_dir):
        for filename in filenames:
            if filename.endswith('.md'):
                file_path = os.path.join(dirpath, filename)
                with open(file_path, 'r', encoding='utf-8') as file:
                    content = file.read()
                
                # Replace the old path with the new path
                new_content, n = img_path_regex.subn(r'![\1](/\3)', content)
                
                if n > 0: # If there were replacements
                    with open(file_path, 'w', encoding='utf-8') as file:
                        file.write(new_content)
                    print(f"Updated image paths in {file_path}")

root_directory = "./content/posts/Programming Languages"
replace_image_path(root_directory)
