export interface AskRequest {
  question: string;
  provider: string;
  topK: number;
  lang?: string;
}

export interface AskSource {
  chunkId: number;
  title: string;
  provider: string;
  sourceUrl?: string;
}

export interface AskResponse {
  answer: string;
  provider: string;
  sources: AskSource[];
}

export interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  sources?: AskSource[];
}
