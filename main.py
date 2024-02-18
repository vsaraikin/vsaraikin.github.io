import os

from markdown_it import MarkdownIt
from mdformat.renderer import MDRenderer
from mdit_py_plugins.front_matter import front_matter_plugin
from mdit_py_plugins.footnote import footnote_plugin

def generate_page(html_text, title, description):
    return f"""<title>{title}</title>
<meta name="description" content="{description}">
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="style/github-markdown.css">
<link rel="stylesheet" href="style/format.css">
<article class="markdown-body">
{html_text}
</article>
"""

input_folder = './knowledge'
output_folder = './notes'

def md_to_html(page_params: dict[str]):
    file = page_params['file']
    
    with open(f'{input_folder}/{file}', 'r') as f:
        text = f.read()
        
    md = (
        MarkdownIt('commonmark' ,{'breaks':True,'html':True})
        .use(front_matter_plugin)
        .use(footnote_plugin)
        .enable('table')
    )

    html_text = md.render(text)
    output = generate_page(html_text, page_params['title'], page_params['description'])

    if not os.path.exists(output_folder):
        raise ValueError(f"path does not exist! {output_folder}")


    with open(f'{output_folder}/{file[:-3]}.html', 'w') as f:
        f.write(output)

files = [{'file': f'brokers.md', 'title': 'Message Queue', 'description': 'message queues'}]

for file in files:
    md_to_html(file)