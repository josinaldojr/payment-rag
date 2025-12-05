import React, { useState } from "react";
import type { AskRequest, AskResponse, Message, AskSource } from "./types";

const RAW_API_URL = (import.meta as any).env?.VITE_API_URL || "http://localhost:8080";
const API_URL = RAW_API_URL.replace(/\/+$/, "");

export const App: React.FC = () => {
  const [lang, setLang] = useState<"auto"|"pt"|"en"|"es">("auto");

  const [provider, setProvider] = useState<"rede" | "entrepay">("rede");
  const [question, setQuestion] = useState("");
  const [topK, setTopK] = useState(5);
  const [messages, setMessages] = useState<Message[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleAsk = async () => {
    const trimmed = question.trim();
    if (!trimmed || loading) return;

    setError(null);

    const userMessage: Message = {
      id: crypto.randomUUID(),
      role: "user",
      content: trimmed
    };

    setMessages((prev) => [...prev, userMessage]);
    setQuestion("");
    setLoading(true);

    try {
      const payload: AskRequest = {
        question: trimmed,
        provider,
        topK,
        lang
      };

      const res = await fetch(`${API_URL}/ask`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });

      if (!res.ok) {
        const txt = await res.text().catch(() => "");
        throw new Error(
          `API error (${res.status}) ${txt ? `- ${txt}` : ""}`
        );
      }

      console.log("Response received from API", res);
      const data = (await res.json()) as AskResponse;

      const assistantMessage: Message = {
        id: crypto.randomUUID(),
        role: "assistant",
        content: data.answer || "No answer returned.",
        sources: data.sources || []
      };

      setMessages((prev) => [...prev, assistantMessage]);
    } catch (err: any) {
      console.error(err);
      setError(err.message || "Unexpected error");
      const assistantMessage: Message = {
        id: crypto.randomUUID(),
        role: "assistant",
        content:
          "Something went wrong calling the RAG API. Please verify the backend is running and try again."
      };
      setMessages((prev) => [...prev, assistantMessage]);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown: React.KeyboardEventHandler<HTMLTextAreaElement> = (
    e
  ) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleAsk();
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col">
      <header className="border-b border-slate-800 bg-slate-950/80 backdrop-blur">
      <div className="flex items-center gap-2 bg-slate-900/80 border border-slate-700 rounded-full px-2 py-1 text-xs">
        <span className="text-slate-400">Idioma</span>
        <select
          value={lang}
          onChange={(e) => setLang(e.target.value as any)}
          className="bg-transparent outline-none"
        >
          <option value="auto">Auto</option>
          <option value="pt">Português</option>
          <option value="en">English</option>
          <option value="es">Español</option>
        </select>
      </div>
        <div className="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">
              Payment Gateway RAG Assistant
            </h1>
            <p className="text-xs text-slate-400">
              Faça perguntas técnicas sobre integrações usando documentação
              indexada (Gemini + pgvector).
            </p>
          </div>
          <div className="flex flex-col items-end gap-1">
            <span className="text-[10px] uppercase text-slate-500">
              Provider
            </span>
            <div className="flex items-center gap-2 bg-slate-900/80 border border-slate-700 rounded-full px-1 py-1 text-xs">
              <button
                className={`px-3 py-1 rounded-full ${
                  provider === "rede"
                    ? "bg-emerald-500 text-slate-950 font-semibold"
                    : "text-slate-400 hover:text-slate-100"
                }`}
                onClick={() => setProvider("rede")}
              >
                e-Rede
              </button>
              <button
                className={`px-3 py-1 rounded-full ${
                  provider === "entrepay"
                    ? "bg-emerald-500 text-slate-950 font-semibold"
                    : "text-slate-500 cursor-not-allowed"
                }`}
                onClick={() => {
                  setProvider("entrepay");
                }}
              >
                Entrepay
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="flex-1">
        <div className="max-w-5xl mx-auto px-4 py-4 flex flex-col h-[calc(100vh-140px)]">
          <div className="flex items-center justify-between mb-3 gap-4">
            <div className="flex items-center gap-3 text-xs text-slate-400">
              <span className="px-2 py-1 rounded-full bg-slate-900/80 border border-slate-800">
                Backend: <span className="text-emerald-400">{API_URL}</span>
              </span>
              <span className="px-2 py-1 rounded-full bg-slate-900/80 border border-slate-800">
                TopK:{" "}
                <input
                  type="range"
                  min={3}
                  max={12}
                  value={topK}
                  onChange={(e) => setTopK(Number(e.target.value))}
                  className="align-middle"
                />{" "}
                <span className="ml-1 text-emerald-400">{topK}</span>
              </span>
            </div>
            {error && (
              <div className="text-xs text-red-400">
                {error}
              </div>
            )}
          </div>

          <div className="flex-1 overflow-y-auto space-y-3 rounded-2xl bg-slate-950/80 border border-slate-800 p-4">
            {messages.length === 0 && (
              <div className="h-full flex flex-col items-center justify-center text-center text-slate-500 text-sm gap-2">
                <p>
                  Comece perguntando algo como:
                </p>
                <code className="bg-slate-900 border border-slate-800 px-3 py-2 rounded-lg text-emerald-300 text-xs">
                  How do I create a 3DS transaction with e-Rede?
                </code>
                <code className="bg-slate-900 border border-slate-800 px-3 py-2 rounded-lg text-emerald-300 text-xs">
                  What are the possible return codes for card brand responses?
                </code>
              </div>
            )}

            {messages.map((m) => (
              <MessageBubble key={m.id} msg={m} />
            ))}

            {loading && (
              <div className="flex items-center gap-2 text-xs text-slate-400">
                <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                Consulting documentation and Gemini...
              </div>
            )}
          </div>

          <div className="mt-4">
            <div className="flex gap-2 items-end">
              <textarea
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                onKeyDown={handleKeyDown}
                rows={2}
                placeholder={
                  provider === "rede"
                    ? "Pergunte algo sobre a API e-Rede, ex: How do I create a 3DS transaction?"
                    : "Pergunte algo sobre o gateway selecionado..."
                }
                className="flex-1 resize-none rounded-2xl bg-slate-950 border border-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/70 focus:border-emerald-500/60"
              />
              <button
                onClick={handleAsk}
                disabled={loading || !question.trim()}
                className={`px-4 py-2 rounded-2xl text-sm font-semibold transition
                  ${
                    loading || !question.trim()
                      ? "bg-slate-800 text-slate-500 cursor-not-allowed"
                      : "bg-emerald-500 text-slate-950 hover:bg-emerald-400"
                  }`}
              >
                {loading ? "Enviando..." : "Perguntar"}
              </button>
            </div>
            <p className="mt-1 text-[10px] text-slate-500">
              Enter para enviar, Shift+Enter para quebrar linha. As respostas
              usam apenas a documentação indexada para <b>{provider}</b>.
            </p>
          </div>
        </div>
      </main>
    </div>
  );
};

const MessageBubble: React.FC<{ msg: Message }> = ({ msg }) => {
  const isUser = msg.role === "user";
  return (
    <div
      className={`flex ${
        isUser ? "justify-end" : "justify-start"
      } text-sm`}
    >
      <div
        className={`max-w-[80%] rounded-2xl px-3 py-2 border backdrop-blur ${
          isUser
            ? "bg-emerald-500 text-slate-950 border-emerald-400"
            : "bg-slate-950/90 text-slate-100 border-slate-800"
        }`}
      >
        <div className="whitespace-pre-wrap">{msg.content}</div>
        {!isUser && msg.sources && msg.sources.length > 0 && (
          <div className="mt-2 border-t border-slate-800 pt-1">
            <div className="text-[9px] uppercase text-slate-500 mb-1">
              Sources
            </div>
            <div className="flex flex-wrap gap-1">
              {msg.sources.map((s: AskSource) => (
                <a
                  key={s.chunkId + s.title}
                  href={s.sourceUrl || "#"}
                  target={s.sourceUrl ? "_blank" : "_self"}
                  rel="noreferrer"
                  className={`text-[9px] px-2 py-1 rounded-full border ${
                    s.sourceUrl
                      ? "border-emerald-500/40 text-emerald-300 hover:bg-emerald-500/10"
                      : "border-slate-700 text-slate-400"
                  }`}
                >
                  {s.title || `Chunk ${s.chunkId}`}
                </a>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
