import os
import base64
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
from cryptography.hazmat.primitives import hashes

def derive_key(password: str, salt: bytes, iterations: int = 480000) -> bytes:
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=salt,
        iterations=iterations,
    )
    return kdf.derive(password.encode())

def encrypt_note(master_password: str, plaintext: str):
    salt = os.urandom(16)
    nonce = os.urandom(12)
    key = derive_key(master_password, salt)
    aesgcm = AESGCM(key)
    ciphertext = aesgcm.encrypt(nonce, plaintext.encode('utf-8'), None)
    
    return {
        "ciphertext": base64.b64encode(ciphertext).decode('utf-8'),
        "salt": base64.b64encode(salt).decode('utf-8'),
        "nonce": base64.b64encode(nonce).decode('utf-8')
    }

def decrypt_note(master_password: str, salt_b64: str, nonce_b64: str, ciphertext_b64: str, iterations: int):
    salt = base64.b64decode(salt_b64)
    nonce = base64.b64decode(nonce_b64)
    ciphertext = base64.b64decode(ciphertext_b64)
    
    key = derive_key(master_password, salt, iterations)
    aesgcm = AESGCM(key)
    plaintext = aesgcm.decrypt(nonce, ciphertext, None)
    
    return plaintext.decode('utf-8')