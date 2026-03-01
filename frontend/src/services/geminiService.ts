import { GoogleGenAI, Type } from "@google/genai";

const getAiClient = () => {
  const apiKey = import.meta.env.VITE_GEMINI_API_KEY || '';
  // We handle the missing key gracefully in the UI, assuming environment variable is set for production
  return new GoogleGenAI({ apiKey });
};

export const analyzeCsvData = async (csvContent: string) => {
  if (!import.meta.env.VITE_GEMINI_API_KEY) {
    console.warn("API Key missing, skipping AI analysis");
    return {
      suggestions: ["API Key not configured. Connect Gemini to analyze network conflicts."],
      risks: [],
      isValid: true
    };
  }

  const ai = getAiClient();

  try {
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