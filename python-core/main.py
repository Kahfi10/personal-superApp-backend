from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware # Tambahkan ini
from pydantic import BaseModel
import psycopg2
from psycopg2.extras import RealDictCursor
from security import encrypt_note, decrypt_note
from uuid import uuid4
from typing import Optional

# Inisialisasi Aplikasi FastAPI
app = FastAPI(title="SuperApp - Security Core API")

# --- TAMBAHKAN BLOK CORS INI ---
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"], # Mengizinkan semua port (termasuk 5173 milik React)
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)
DB_CONFIG = {
    "dbname": "postgres",
    "user": "postgres",
    "password": "password123",
    "host": "localhost",
    "port": "5432"
}

class NoteCreate(BaseModel):
    user_id: Optional[str] = None  # Jika kosong, akan auto-generate dengan UUID
    title: str
    content: str
    master_password: str

class NoteDecrypt(BaseModel):
    master_password: str

def get_db_connection():
    return psycopg2.connect(**DB_CONFIG, cursor_factory=RealDictCursor)

@app.post("/notes/")
def create_secure_note(note: NoteCreate):
    # Auto-generate User ID jika tidak diberikan
    user_id = note.user_id if note.user_id else str(uuid4())
    encrypted_data = encrypt_note(note.master_password, note.content)
    conn = get_db_connection()
    cursor = conn.cursor()
    try:
        query = "INSERT INTO notes (user_id, title, content, encryption_iv, encryption_tag) VALUES (%s, %s, %s, %s, %s) RETURNING id;"
        cursor.execute(query, (user_id, note.title, encrypted_data["ciphertext"], encrypted_data["nonce"], encrypted_data["salt"]))
        new_note_id = cursor.fetchone()['id']
        conn.commit()
        return {"message": "Catatan aman berhasil disimpan!", "note_id": new_note_id, "user_id": user_id}
    except Exception as e:
        conn.rollback()
        raise HTTPException(status_code=500, detail=f"Database error: {str(e)}")
    finally:
        cursor.close()
        conn.close()

@app.post("/notes/{note_id}/decrypt")
def read_secure_note(note_id: str, request: NoteDecrypt):
    conn = get_db_connection()
    cursor = conn.cursor()
    try:
        cursor.execute("SELECT title, content, encryption_iv, encryption_tag FROM notes WHERE id = %s;", (note_id,))
        result = cursor.fetchone()
        if not result:
            raise HTTPException(status_code=404, detail="Catatan tidak ditemukan")
        try:
            plaintext = decrypt_note(request.master_password, result['encryption_tag'], result['encryption_iv'], result['content'], 480000)
        except Exception:
            raise HTTPException(status_code=401, detail="Master password salah atau data korup.")
        return {"id": note_id, "title": result['title'], "decrypted_content": plaintext}
    finally:
        cursor.close()
        conn.close()