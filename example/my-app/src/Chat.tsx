import {useEffect, useState} from "react";
import './Chat.css';
import {ChatCMD, ChatResult, dediscriminateChatCMD} from "./workflow/main";

type Message = {
    text: string,
    type: "user" | "system"
}

export function Chat(props: { name: string }) {
    const [messages, setMessages] = useState<Message[]>([]);

    useEffect(() => {
        let lastMessage = messages[messages.length - 1];
        if (lastMessage && lastMessage.type === "user") {
            let cmd: ChatCMD = {
                "main.UserMessage": {
                    "Message": lastMessage.text,
                },
            };
            fetch('http://localhost:8080/message', {
                method: 'POST',
                body: JSON.stringify(dediscriminateChatCMD(cmd)),
            }).then(res => res.json() as Promise<ChatResult>)
                .then(data => {
                    if ( "main.SystemResponse" in data) {
                        if (data["main.SystemResponse"]["Message"] as string !== "") {
                            setMessages([...messages, {
                                type: "system",
                                text: data["main.SystemResponse"]["Message"] as string
                            }])
                        }

                        let toolCalls = data["main.SystemResponse"]["ToolCalls"] || [];
                        toolCalls.forEach((toolCall: any) => {
                            switch (toolCall.Function.Name) {
                                case "list_workflows":

                            }
                        })
                    }
                });
        }

    }, [messages]);

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