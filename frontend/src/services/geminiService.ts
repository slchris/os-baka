// Gemini AI Service — dynamically loaded to avoid bundling @google/genai
// when the feature is not used (no API key configured).

import { GeminiAnalysisResult } from '../types';

export const analyzeCsvData = async (csvContent: string): Promise<GeminiAnalysisResult> => {
  if (!import.meta.env.VITE_GEMINI_API_KEY) {
    console.warn("API Key missing, skipping AI analysis");
    return {
      suggestions: ["API Key not configured. Set VITE_GEMINI_API_KEY to enable AI analysis."],
      risks: [],
      isValid: true
    };
  }

  try {
    // Dynamic import — only loads @google/genai when actually needed
    const { GoogleGenAI, Type } = await import("@google/genai");

    const ai = new GoogleGenAI({ apiKey: import.meta.env.VITE_GEMINI_API_KEY });

    const response = await ai.models.generateContent({
      model: 'gemini-2.5-flash',
      contents: `Analyze the following CSV data representing network nodes (MAC, IP, Hostname). 
      Check for:
      1. Duplicate IPs or MAC addresses.
      2. Invalid IP formats.
      3. Hostname naming convention consistency.
      
      CSV Data:
      ${csvContent.substring(0, 2000)}... (truncated)
      
      Return a JSON object with 'suggestions' (array of strings), 'risks' (array of strings), and 'isValid' (boolean).`,
      config: {
        responseMimeType: "application/json",
        responseSchema: {
          type: Type.OBJECT,
          properties: {
            suggestions: {
              type: Type.ARRAY,
              items: { type: Type.STRING },
            },
            risks: {
              type: Type.ARRAY,
              items: { type: Type.STRING },
            },
            isValid: { type: Type.BOOLEAN },
          },
        },
      },
    });

    const text = response.text;
    if (text) {
      return JSON.parse(text);
    }
    throw new Error("Empty response from AI");

  } catch (error) {
    console.error("Gemini Analysis Failed:", error);
    return {
      suggestions: ["AI Analysis failed. Please check format manually."],
      risks: ["Could not verify data integrity via AI."],
      isValid: true // Default to true to allow manual override
    };
  }
};