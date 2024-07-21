#!/bin/python3

import mimetypes

def extension_to_mime(extension):
    t = mimetypes.guess_type('file.'+extension, False)

    return t[0] if t else None

if __name__ == '__main__':
    import sys
    if len(sys.argv) < 2:
        print('Usage: extension_to_mime.py [extensions]')
        sys.exit(1)
    
    extensions = []
    for extension in sys.argv[1:]:
        t = extension_to_mime(extension)
        if t:
            extensions.append(t)
    
    print('[]string{"'+'", "'.join(extensions)+'"}')