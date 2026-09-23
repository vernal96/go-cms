#!/usr/bin/env python3
"""Create a private local environment with unique credentials; never overwrite."""
from pathlib import Path
import os
import secrets

path = Path('.env')
if path.exists():
    print('Using existing .env')
else:
    text = Path('.env.example').read_text()
    values = {
        'POSTGRES_PASSWORD': secrets.token_urlsafe(32),
        'JWT_SIGNING_KEY': secrets.token_urlsafe(48),
        'FILES_PRIVATE_SIGNING_KEY': secrets.token_urlsafe(48),
    }
    text = '\n'.join(f'{line.split("=", 1)[0]}={values[line.split("=", 1)[0]]}'
                     if line.split('=', 1)[0] in values else line
                     for line in text.splitlines()) + '\n'
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as output:
        output.write(text)
    print('Created .env with unique credentials; development seed is disabled')
