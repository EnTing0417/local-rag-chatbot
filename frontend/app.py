import gradio as gr
import requests

API_URL = "http://backend:8000/chat"

def chat_with_bot(query):
    response = requests.post(API_URL, json={"query": query})
    return response.json()["response"]

iface = gr.Interface(fn=chat_with_bot, inputs="text", outputs="text", title="RAG Chatbot")

iface.launch(server_name="0.0.0.0", server_port=7860)