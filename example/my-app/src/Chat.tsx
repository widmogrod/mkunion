import {useEffect, useState} from "react";
import './Chat.css';
import {ChatCMD, ChatResult} from "./workflow/github_com_widmogrod_mkunion_exammple_my-app";

type Message = {
    text: string,
    type: "user" | "system"
}


interface ChatParams {
    props: {
        name: string;
        onFunctionCall?: (func: { Name: string, Arguments: string }) => void
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

function getToolCalls(result: ChatResult): any[] {
    if ("main.SystemResponse" in result) {
        return result["main.SystemResponse"]["ToolCalls"] || []
    } else if ("main.ChatResponses" in result) {
        const responses = result["main.ChatResponses"]["Responses"] || [];
        return responses.flatMap((response: ChatResult) => {
            return getToolCalls(response)
        }).filter((toolCall: any) => toolCall)
    }
    return []
}

export function Chat({props}: ChatParams) {
    const [messages, setMessages] = useState<Message[]>([
        {type: "system", text: "Hello, " + props.name + "!"},
        {type: "system", text: "What can I do for you?"},
    ]);

    useEffect(() => {
        let lastMessage = messages[messages.length - 1];
        if (!lastMessage || lastMessage.type !== "user") {
            return
        }

        let cmd: ChatCMD = {
            "$type": "main.UserMessage",
            "main.UserMessage": {
                "Message": lastMessage.text,
            },
        };

        fetch('http://localhost:8080/message', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(cmd),
        })
            .then(res => res.json() as Promise<ChatResult>)
            .then(data => {
                    setMessages([...messages, ...mapChatResultToMessages(data)])

                    let toolCalls = getToolCalls(data)
                    toolCalls.forEach((toolCall: any) => {
                        props.onFunctionCall && props.onFunctionCall(toolCall.Function);
                    })
            });

    }, [messages, props]);

    return <div className="chat-window">
        <ChatHistory messages={messages}/>
        <ChatInput onSubmit={(message) => setMessages([...messages, {type: "user", text: message}])}/>
    </div>;
}

export function ChatHistory(props: { messages: Message[] }) {
    return <div className="chat-history">
        {props.messages.map((message, index) =>
            <div key={index} className={"chat-message " + message.type}>
                {message.text}
            </div>
        )}
    </div>;
}

export function ChatInput(props: { onSubmit: (message: string) => void }) {
    return <form className="chat-input" onSubmit={event => {
        event.preventDefault();
        const input = event.currentTarget.querySelector("input") as HTMLInputElement;
        props.onSubmit(input.value);
        input.value = "";
    }}>
        <input type="text"/>
    </form>;
}