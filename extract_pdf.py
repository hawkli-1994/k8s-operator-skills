#!/usr/bin/env python3
import PyPDF2
import sys
import os

def extract_text_from_pdf(pdf_path, output_prefix='part', chunk_size=50):
    """Extract text from PDF and split into chunks"""

    with open(pdf_path, 'rb') as file:
        pdf_reader = PyPDF2.PdfReader(file)
        total_pages = len(pdf_reader.pages)

        print(f"Total pages: {total_pages}")

        chunk_num = 1
        start_page = 0

        while start_page < total_pages:
            end_page = min(start_page + chunk_size, total_pages)

            output_file = f'/mnt/d/k8s-operator-skills/docs/{output_prefix}{chunk_num}.md'

            with open(output_file, 'w', encoding='utf-8') as out:
                out.write(f"# Pages {start_page + 1}-{end_page}\n\n")

                for page_num in range(start_page, end_page):
                    page = pdf_reader.pages[page_num]
                    try:
                        text = page.extract_text()
                        out.write(f"## Page {page_num + 1}\n\n{text}\n\n")
                    except Exception as e:
                        out.write(f"## Page {page_num + 1}\n\n[Error extracting text: {e}]\n\n")

                print(f"Created {output_file} (pages {start_page + 1}-{end_page})")

            chunk_num += 1
            start_page = end_page

        print(f"\nExtraction complete! Created {chunk_num - 1} files.")

if __name__ == '__main__':
    pdf_path = '/mnt/d/k8s-operator-skills/Kubernetes编程_14981181.PDF'
    extract_text_from_pdf(pdf_path, chunk_size=50)
