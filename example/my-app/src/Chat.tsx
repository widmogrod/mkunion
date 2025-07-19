import { useState } from "react";
import { Input } from "./components/ui/input";
import { Button } from "./components/ui/button";
import { cn } from "./lib/utils";
import * as openai from './workflow/github_com_sashabaranov_go-openai'
import { ChatCMD, ChatResult } from "./workflow/github_com_widmogrod_mkunion_exammple_my-app";
import { assertNever } from "./utils/type-helpers";

type Message = {
    text: string,
    type: "user" | "system"
}

interface ChatParams {
    props: {
        name: string;
        onFunctionCall?: (func: openai.FunctionCall) => void
    };
}

function mapChatResultToMessages(result: ChatResult): Message[] {
    if ("main.SystemResponse" in result) {
        return [{
            type: "system",
            text: result["main.SystemResponse"]["Message"] as string || "acknowledged system"
        }]
    } else if ("main.UserResponse" in result) {
        return [{
            type: "user",
            text: result["main.UserResponse"]["Message"] as string || "acknowledged user"
        }]
    } else if ("main.ChatResponses" in result) {
        const responses = result["main.ChatResponses"]["Responses"] || [];
        return responses.flatMap((response: ChatResult) => {
            return mapChatResultToMessages(response)
        })
    }
    return [
        {
            type: "system",
            text: "acknowledged default"
        },
    ]
}

function getToolCalls(result: ChatResult): openai.ToolCall[] {
    const resultType = result.$type;
    if (!resultType) {
        console.error('getToolCalls: result.$type is undefined', result);
        return [];
    }

    switch (resultType) {
        case "main.SystemResponse":
            return result["main.SystemResponse"].ToolCalls || [];

        case "main.UserResponse":
            // UserResponse doesn't have tool calls
            return [];

        case "main.ChatResponses":
            const responses = result["main.ChatResponses"]["Responses"] || [];
            return responses.flatMap((response: ChatResult) => {
                return getToolCalls(response)
            }).filter((toolCall: any) => toolCall)
        default:
            return assertNever(resultType);
    }
}

export function Chat({ props }: ChatParams) {
    const [messages, setMessages] = useState<Message[]>([
        { type: "system", text: "Hello, " + props.name + "!" },
        { type: "system", text: "What can I do for you?" },
    ]);
    const [input, setInput] = useState("");
    const [loading, setLoading] = useState(false);

    const sendMessage = async (message: string) => {
        if (!message.trim() || loading) return;

        setMessages(prev => [...prev, { type: "user", text: message }]);
        setInput("");
        setLoading(true);

        try {
            let cmd: ChatCMD = {
                "$type": "main.UserMessage",
                "main.UserMessage": {
                    "Message": message,
                },
            };

            const response = await fetch('http://localhost:8080/message', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(cmd),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json() as ChatResult;
            setMessages(prev => [...prev, ...mapChatResultToMessages(data)]);

            let toolCalls = getToolCalls(data);
            toolCalls.forEach((toolCall: openai.ToolCall) => {
                props.onFunctionCall && toolCall.Function && props.onFunctionCall(toolCall.Function);
            });
        } catch (error) {
            console.error('Chat error:', error);
            setMessages(prev => [...prev, { 
                type: "system", 
                text: "Error: Unable to send message. Please check if the server is running." 
            }]);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="flex flex-col h-[400px] space-y-4">
            <div className="flex-1 overflow-y-auto rounded-lg border bg-background p-4 space-y-2">
                {messages.map((message, index) => (
                    <div
                        key={index}
                        className={cn(
                            "rounded-lg px-3 py-2 max-w-[80%]",
                            message.type === "user" 
                                ? "ml-auto bg-primary text-primary-foreground" 
                                : "mr-auto bg-muted"
                        )}
                    >
                        <p className="text-sm">{message.text}</p>
                    </div>
                ))}
            </div>
            <form 
                onSubmit={(e) => {
                    e.preventDefault();
                    sendMessage(input);
                }}
                className="flex gap-2"
            >
                <Input
                    type="text"
                    placeholder="Type a message..."
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    disabled={loading}
                    className="flex-1"
                />
                <Button type="submit" disabled={loading || !input.trim()}>
                    Send
                </Button>
            </form>
        </div>
    );
}