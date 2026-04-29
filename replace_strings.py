import os

replacements = {
    "github.com/shark-auth/shark": "github.com/shark-auth/shark",
    "sharkauth.com": "sharkauth.com"
}

exclude_dirs = {".git", "node_modules", ".claude", ".agent", ".gemini", ".gstack", "bin"}
exclude_files = {"shark.exe", "shark"}

def perform_replacements():
    for root, dirs, files in os.walk("."):
        dirs[:] = [d for d in dirs if d not in exclude_dirs]
        for file in files:
            if file in exclude_files:
                continue
            
            file_path = os.path.join(root, file)
            
            # Skip non-text files by attempting to read as utf-8
            try:
                with open(file_path, "r", encoding="utf-8") as f:
                    content = f.read()
            except (UnicodeDecodeError, PermissionError):
                continue
                
            new_content = content
            changed = False
            for old, new in replacements.items():
                if old in new_content:
                    new_content = new_content.replace(old, new)
                    changed = True
            
            if changed:
                print(f"Updating {file_path}")
                with open(file_path, "w", encoding="utf-8") as f:
                    f.write(new_content)

if __name__ == "__main__":
    perform_replacements()
