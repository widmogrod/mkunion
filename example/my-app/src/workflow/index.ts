export type ToolCall = {
    index?: number;
    id: string;
    type: string;
    function: FunctionCall;
}

export type FunctionCall = {
    name?: string;
    arguments?: string;
}