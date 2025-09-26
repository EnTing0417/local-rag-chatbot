from flask import Flask, request, jsonify, render_template
from rag import retrieve_context, query_ollama

app = Flask(__name__)

@app.route("/")
def index():
    return render_template("index.html")

@app.route("/chat", methods=["POST"])
def chat():
    user_input = request.json.get("message", "")
    context = retrieve_context(user_input)
    prompt = f"{context}\n\nUser: {user_input}\nAssistant:"
    response = query_ollama(prompt)
    return jsonify({"response": response})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000)