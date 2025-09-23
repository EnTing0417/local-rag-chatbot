import gradio as gr
import requests
import subprocess
import os
import shutil

# Change this to match your Go server URL/port
SERVER_URL = "http://localhost:8080/chat"

# Available models (must exist in your Ollama environment)
MODELS = ["gemma:2b", "llama2", "mistral"]

def chat_with_bot(user_message, history, model):
    try:
        resp = requests.post(SERVER_URL, json={"query": user_message, "model": model})
        resp.raise_for_status()
        answer = resp.json().get("answer", "⚠️ No response from server")
    except Exception as e:
        answer = f"Error: {e}"
    history.append((user_message, answer))
    return history, history

def upload_and_ingest(file):
    if not file:
        return "No file uploaded."
    dest_path = os.path.join("docs", os.path.basename(file.name))
    shutil.copy(file.name, dest_path)

    try:
        # Run the Python ingest pipeline
        result = subprocess.run(
            ["python", "ingest.py"], capture_output=True, text=True
        )
        if result.returncode != 0:
            return f"Ingest failed: {result.stderr}"
        return f"✅ Ingest complete for {os.path.basename(file.name)}"
    except Exception as e:
        return f"Error running ingest: {e}"

with gr.Blocks() as demo:
    gr.Markdown("# 💬 Local RAG Chatbot")

    with gr.Row():
        with gr.Column(scale=2):
            chatbot = gr.Chatbot(height=400)
            msg = gr.Textbox(label="Type your question")
            model_dropdown = gr.Dropdown(
                choices=MODELS, value=MODELS[0], label="Select Model"
            )
            clear = gr.Button("Clear Chat")

        with gr.Column(scale=1):
            file_upload = gr.File(label="Upload a document (.txt)", type="filepath")
            status = gr.Textbox(label="Ingest Status", interactive=False)

    msg.submit(chat_with_bot, [msg, chatbot, model_dropdown], [chatbot, chatbot])
    clear.click(lambda: ([], []), None, [chatbot, chatbot])
    file_upload.upload(upload_and_ingest, [file_upload], [status])

if __name__ == "__main__":
    demo.launch(server_name="0.0.0.0", server_port=7860)
