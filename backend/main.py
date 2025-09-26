from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from rag import get_rag_response

app = FastAPI()
app.add_middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"], allow_headers=["*"])

@app.post("/chat")
async def chat(request: Request):
    data = await request.json()
    query = data.get("query", "")
    response = get_rag_response(query)
    return {"response": response}