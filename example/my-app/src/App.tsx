import React, {createContext, useEffect, useState} from 'react';
import './App.css';
import * as openai from './workflow/github_com_sashabaranov_go-openai'
import * as schemaless from './workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as workflow from './workflow/github_com_widmogrod_mkunion_x_workflow'
import * as predicate from "./workflow/github_com_widmogrod_mkunion_x_storage_predicate";
import * as schema from "./workflow/github_com_widmogrod_mkunion_x_schema";
import {Chat} from "./Chat";
import {PaginatedTable} from "./component/PaginatedTable";

function flowCreate(flow: workflow.Flow) {
    return fetch('http://localhost:8080/flow', {
        method: 'POST',
        body: JSON.stringify(flow),
    })
        .then(res => res.text())
        .then(data => console.log("save-flow-result", data))
}

function flowToStringFromWorkflow(flow: workflow.Workflow) {
    return fetch('http://localhost:8080/workflow-to-str', {
        method: 'POST',
        body: JSON.stringify(flow),
    })
        .then(res => res.text())
}

function flowToStringFromRunID(runID: string) {
    return fetch(`http://localhost:8080/workflow-to-str-from-run/${runID}`, {
        method: 'GET',
    })
        .then(res => res.text())
}

type ListProps<T> = {
    baseURL?: string,
    path?: string,
    sort?: {
        [key: string]: boolean
    }
    limit?: number,
    where?: predicate.WherePredicates

    prevPage?: string,
    nextPage?: string,
}

function storageList<T>(input: ListProps<T>): Promise<schemaless.PageResult<schemaless.Record<T>>> {
    let url = input.baseURL || 'http://localhost:8080/'
    url = url + input.path

    return fetch(url, {
        method: 'POST',
        body: JSON.stringify({
            Limit: input.limit || 30,
            Sort: input.sort && Object.keys(input.sort).map((key) => {
                return {
                    Field: key,
                    Descending: input.sort?.[key],
                }
            }),
            Where: input.where,
            After: input.nextPage,
            Before: input.prevPage,
        } as schemaless.FindingRecords<schemaless.Record<T>>),
    })
        .then(res => res.json())
        .then(data => data as schemaless.PageResult<schemaless.Record<T>>)
}

function listStates(input?: ListProps<workflow.State>) {
    return storageList<workflow.State>({
        ...input,
        path: "states",
    })
}

function listFlows(input?: ListProps<workflow.Flow>) {
    return storageList<workflow.Flow>({
        ...input,
        path: "flows",
    })
}

type UpdatingProps<T> = {
    baseURL?: string,
    path?: string,
    data: schemaless.UpdateRecords<schemaless.Record<T>>,
}

function updatingRecords<T>(input: UpdatingProps<T>) {
    let url = input.baseURL || 'http://localhost:8080/'
    url = url + input.path

    return fetch(url, {
        method: 'POST',
        body: JSON.stringify(input.data),
    })
        .then(res => {
        })
}

function deleteFlows(flows: schemaless.Record<workflow.Flow>[]) {
    let deleting = {} as { [key: string]: schemaless.Record<workflow.Flow> }
    flows.forEach((flow) => {
        if (!flow.ID) {
            return
        }

        deleting[flow.ID] = flow
    })

    return updatingRecords<workflow.Flow>({
        path: "flows-updating",
        data: {
            Deleting: deleting,
        }
    })
}

function deleteStates(states: schemaless.Record<workflow.State>[]) {
    let deleting = {} as { [key: string]: schemaless.Record<workflow.State> }
    states.forEach((flow) => {
        if (!flow.ID) {
            return
        }

        deleting[flow.ID] = flow
    })

    return updatingRecords<workflow.State>({
        path: "state-updating",
        data: {
            Deleting: deleting,
        }
    })
}

function runFlow(flowID: string, input: string, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "$type": "workflow.Run",
        "workflow.Run": {
            Flow: {
                "$type": "workflow.FlowRef",
                "workflow.FlowRef": {
                    FlowID: flowID,
                }
            },
            Input: {
                "schema.String": input,
            },
        }
    }
    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => {
            onData && onData(data)
        })

}

function runHelloWorldWorkflow(input: string, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "$type": "workflow.Run",
        "workflow.Run": {
            Flow: {
                "$type": "workflow.Flow",
                "workflow.Flow": {
                    Name: "hello_world",
                    Arg: "input",
                    Body: [
                        {
                            "$type": "workflow.Choose",
                            "workflow.Choose": {
                                ID: "choose1",
                                If: {
                                    "$type": "workflow.Compare",
                                    "workflow.Compare": {
                                        Operation: "=",
                                        Left: {
                                            "$type": "workflow.GetValue",
                                            "workflow.GetValue": {
                                                Path: "input",
                                            }
                                        },
                                        Right: {
                                            "$type": "workflow.SetValue",
                                            "workflow.SetValue": {
                                                Value: {
                                                    "schema.String": "666",
                                                },
                                            },
                                        },
                                    }
                                },
                                Then: [
                                    {
                                        "$type": "workflow.End",
                                        "workflow.End": {
                                            ID: "end2",
                                            Result: {
                                                "$type": "workflow.SetValue",
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "Do no evil",
                                                    },
                                                }
                                            },
                                        },
                                    }
                                ],
                            }
                        },
                        {
                            "$type": "workflow.Assign",
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                VarErr: "",
                                Val: {
                                    "$type": "workflow.Apply",
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "concat",
                                        Args: [
                                            {
                                                "$type": "workflow.SetValue",
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "hello ",
                                                    }
                                                }
                                            },
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input",
                                                }
                                            },
                                        ]
                                    }
                                }
                            },
                        },
                        {
                            "$type": "workflow.End",
                            "workflow.End": {
                                ID: "end1",
                                Result: {
                                    "$type": "workflow.GetValue",
                                    "workflow.GetValue": {
                                        Path: "res",
                                    }
                                }
                            }
                        }
                    ],
                },
            },
            Input: {
                "schema.String": input,
            },
        }
    }

    if (cmd?.["workflow.Run"]?.Flow) {
        flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
    }

    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => onData && onData(data))
}

function runErrorWorkflow(input: string, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "$type": "workflow.Run",
        "workflow.Run": {
            Flow: {
                "$type": "workflow.Flow",
                "workflow.Flow": {
                    Name: "do_error",
                    Arg: "input",
                    Body: [
                        {
                            "$type": "workflow.Assign",
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                VarErr: "",
                                Val: {
                                    "$type": "workflow.Apply",
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "concat_error",
                                        Args: [
                                            {
                                                "$type": "workflow.SetValue",
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "hello ",
                                                    }
                                                }
                                            },
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input",
                                                }
                                            },
                                        ]
                                    }
                                }
                            },
                        },
                        {
                            "$type": "workflow.End",
                            "workflow.End": {
                                ID: "end1",
                                Result: {
                                    "$type": "workflow.GetValue",
                                    "workflow.GetValue": {
                                        Path: "res",
                                    }
                                }
                            }
                        }
                    ],
                },
            },
            Input: {
                "schema.String": input,
            },
        }
    }

    if (cmd?.["workflow.Run"]?.Flow) {
        flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
    }

    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => onData && onData(data))
}

function generateImage(imageWidth: number, imageHeight: number, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "$type": "workflow.Run",
        "workflow.Run": {
            Flow: {
                "$type": "workflow.Flow",
                "workflow.Flow": {
                    Name: "generateandresizeimage",
                    Arg: "input",
                    Body: [
                        {
                            "$type": "workflow.Assign",
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                VarErr: "",
                                Val: {
                                    "$type": "workflow.Apply",
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "genimageb64",
                                        Args: [
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input.prompt",
                                                }
                                            },
                                        ]
                                    }
                                }
                            },
                        },
                        {
                            "$type": "workflow.Assign",
                            "workflow.Assign": {
                                ID: "assign2",
                                VarOk: "res_small",
                                VarErr: "",
                                Val: {
                                    "$type": "workflow.Apply",
                                    "workflow.Apply": {
                                        ID: "apply2",
                                        Name: "resizeimgb64",
                                        Args: [
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "res",
                                                }
                                            },
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input.width",
                                                }
                                            },
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input.height",
                                                }
                                            },
                                        ]
                                    }
                                }
                            },
                        },
                        {
                            "$type": "workflow.End",
                            "workflow.End": {
                                ID: "end1",
                                Result: {
                                    "$type": "workflow.GetValue",
                                    "workflow.GetValue": {
                                        Path: "res_small",
                                    }
                                }
                            }
                        }
                    ],
                },
            },
            Input: {
                "schema.Map": {
                    "prompt": {"schema.String": "no text"},
                    "width": {"schema.Number": imageWidth},
                    "height": {"schema.Number": imageHeight},
                },
            },
        }
    }

    if (cmd?.["workflow.Run"]?.Flow) {
        flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
    }

    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then((data: workflow.State) => {
            onData && onData(data)

        })
}

function runContactAwait(imageWidth: number, imageHeight: number, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "$type": "workflow.Run",
        "workflow.Run": {
            Flow: {
                "$type": "workflow.Flow",
                "workflow.Flow": {
                    Name: "concat_await",
                    Arg: "input",
                    Body: [
                        {
                            "$type": "workflow.Assign",
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                VarErr: "",
                                Val: {
                                    "$type": "workflow.Apply",
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "concat",
                                        Args: [
                                            {
                                                "$type": "workflow.SetValue",
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "await hello ",
                                                    }
                                                }
                                            },
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input.prompt",
                                                }
                                            },
                                        ],
                                        Await: {
                                            TimeoutSeconds: 10,
                                        }
                                    }
                                }
                            },
                        },
                        {
                            "$type": "workflow.End",
                            "workflow.End": {
                                ID: "end1",
                                Result: {
                                    "$type": "workflow.GetValue",
                                    "workflow.GetValue": {
                                        Path: "res",
                                    }
                                }
                            }
                        }
                    ],
                },
            },
            Input: {
                "schema.Map": {
                    "prompt": {"schema.String": "no text"},
                    "width": {"schema.Number": imageWidth},
                    "height": {"schema.Number": imageHeight},
                },
            },
        }
    }

    if (cmd?.["workflow.Run"]?.Flow) {
        flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
    }

    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then((data: workflow.State) => {
            onData && onData(data)
        })
}

function submitCallbackResult(callbackID: string, res: schema.Schema, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "$type": "workflow.Callback",
        "workflow.Callback": {
            CallbackID: callbackID,
            Result: res,
        }
    }

    fetch('http://localhost:8080/callback', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then((data: workflow.State) => {
            onData && onData(data)
        })
}

function extractParentRunID(state: workflow.State): string | undefined {
    switch (state.$type) {
        case "workflow.Scheduled":
            switch (state["workflow.Scheduled"].BaseState?.RunOption?.$type) {
                case "workflow.ScheduleRun":
                    return state["workflow.Scheduled"].BaseState?.RunOption?.["workflow.ScheduleRun"].ParentRunID
            }
            break

        case "workflow.ScheduleStopped":
            switch (state["workflow.ScheduleStopped"].BaseState?.RunOption?.$type) {
                case "workflow.ScheduleRun":
                    return state["workflow.ScheduleStopped"].BaseState?.RunOption?.["workflow.ScheduleRun"].ParentRunID
            }
            break
    }

    return undefined
}

function App() {
    const [state, setState] = React.useState({} as workflow.State);
    const [input, setInput] = React.useState("hello");
    const [output, setOutput] = React.useState("" as any);


    const [image, setImage] = React.useState("" as string);
    const [imageWidth, setImageWidth] = React.useState(100 as number);
    const [imageHeight, setImageHeight] = React.useState(100 as number);
    const [selectedFlow, setSelectedFlow] = React.useState("hello_world" as string);

    const setImageFromState = (data: workflow.State) => {
        if ("workflow.Done" in data) {
            if (data["workflow.Done"].Result) {
                let result = data["workflow.Done"].Result
                if ("schema.Binary" in result) {
                    setImage(result["schema.Binary"])
                }
            }
        } else if ("workflow.Error" in data) {
            console.log(data["workflow.Error"])
        }
    }

    return (
        <div className="App">
            <aside>
                <HelloWorldDemo/>

                <form
                    className={"action-section"}
                    onSubmit={(e) => {
                        e.preventDefault()
                        generateImage(imageWidth, imageHeight, (data) => {
                            setImageFromState(data)
                        })
                    }}
                >
                    <h2>Image generation</h2>
                    <input type="number"
                           placeholder="Width"
                           onInput={(e) => setImageWidth(parseInt(e.currentTarget.value))}/>
                    <input type="number"
                           placeholder="Height"
                           onInput={(e) => setImageHeight(parseInt(e.currentTarget.value))}/>
                    <button>
                        Generate image
                    </button>
                </form>

                {/*<form*/}
                {/*    className={"action-section"}*/}
                {/*    onSubmit={(e) => {*/}
                {/*        e.preventDefault()*/}
                {/*        runFlow(selectedFlow, input, (data) => {*/}
                {/*            setImageFromState(data)*/}
                {/*        })*/}
                {/*    }}*/}
                {/*>*/}
                {/*    <h2>Run selected flow</h2>*/}
                {/*    <select value={selectedFlow}*/}
                {/*            onChange={(e) => setSelectedFlow(e.currentTarget.value)}>*/}
                {/*        {flowsData.Items?.map((item) => {*/}
                {/*            return (*/}
                {/*                <option key={item.ID} value={item.ID}>{item.ID}</option>*/}
                {/*            );*/}
                {/*        })}*/}
                {/*    </select>*/}

                {/*    <button>*/}
                {/*        Run selected flow*/}
                {/*    </button>*/}
                {/*</form>*/}

                <form className={"action-section"}>
                    <h2>Async and callback result</h2>
                    <button onClick={(e) => {
                        e.preventDefault()
                        runContactAwait(imageWidth, imageHeight, (data) => {
                            setImageFromState(data)
                        })
                    }
                    }>
                        Run concat await
                    </button>
                </form>

                <form className={"action-section"}>
                    <h2>Schedule run</h2>
                    <SchedguledRun input={input}/>
                </form>

                <form className={"action-section"}>
                    <h2>Invoke function without workflow</h2>
                    <button onClick={(e) => {
                        e.preventDefault()
                        callFunc("concat", [
                            {"schema.String": "hello "},
                            {"schema.String": input},
                        ]).then((data) => {
                            setOutput(JSON.stringify(data))
                        })
                    }}>
                        Call func - Concat with {input}
                    </button>
                </form>

                <div className={"action-section"}>
                    <h2>Chat</h2>

                    <Chat
                        props={{
                            name: "John",
                            onFunctionCall: (x: openai.FunctionCall) => {
                                console.log("onFunctionCall", x);
                                // switch (x.name) {
                                //     case "count_words":
                                //         let args = JSON.parse(x.arguments || "") as ListWorkflowsFn
                                //         console.log(args)
                                //         break
                                //
                                //     case "refresh_states":
                                //         listStates().then(setStatesData)
                                //         break;
                                //
                                //     case "refresh_flows":
                                //         console.log("refresh_flows")
                                //         listFlows().then(setFlowsData)
                                //         break;
                                //
                                //     case "generate_image":
                                //         let args2 = JSON.parse(x.arguments || "") as GenerateImage;
                                //         generateImage(args2?.Width || 100, args2?.Height || 100, (data) => {
                                //             setImageFromState(data)
                                //             listStates().then(setStatesData)
                                //             listFlows().then(setFlowsData)
                                //         })
                                //         break;
                                // }
                            }
                        }}
                    />
                </div>
            </aside>
            <main>
                <table>
                    <tbody>
                    <tr>
                        <td>
                            <PaginatedTable<workflow.Flow>
                                sort={{
                                    "ID": true,
                                }}
                                load={(state) => {
                                    return listFlows({
                                        ...state,
                                    })
                                }}
                                mapData={(data) => {
                                    return <WorkflowToString flow={{
                                        "$type": "workflow.Flow",
                                        "workflow.Flow": data.Data || {}
                                    }}/>
                                }}
                                actions={[
                                    {
                                        name: "Delete selected",
                                        action: (state, ctx) => {
                                            deleteFlows([...Object.values(state.selected)])
                                                .then(() => ctx.refresh())
                                        }
                                    },
                                ]}
                            />

                            <div className={"debug-window"}>
                                <img src={`data:image/jpeg;base64,${image}`} alt=""/>
                                <pre>Func output: {output}</pre>
                                <pre>Workflow output: {JSON.stringify(state, null, 2)} </pre>
                            </div>
                        </td>
                        <td>
                            <PaginatedTable<workflow.State>
                                sort={{
                                    "ID": true,
                                }}
                                load={(state) => {
                                    return listStates({
                                        ...state,
                                    })
                                }}
                                actions={[
                                    {
                                        name: "Delete",
                                        action: (state, ctx) => {
                                            deleteStates([...Object.values(state.selected)])
                                                .then(() => {
                                                    ctx.refresh()
                                                    ctx.clearSelection()
                                                })
                                        }
                                    },
                                    {
                                        name: "Try recover",
                                        action: (state, ctx) => {
                                            const requests = Object.keys(state.selected).map((key) => {
                                                let value = state.selected[key]
                                                if (!value.ID) {
                                                    return
                                                }

                                                return recover(value.ID)
                                            })

                                            Promise.all(requests).then(() => {
                                                ctx.refresh()
                                                ctx.clearSelection()
                                            })
                                        }
                                    }
                                ]}
                                mapData={(input, ctx) => {
                                    if (!input.Data) {
                                        return <div>nothing</div>
                                    }

                                    let data = input.Data

                                    switch (data.$type) {
                                        case "workflow.Done":
                                            let done = data["workflow.Done"]
                                            switch (done.Result?.$type) {
                                                case "schema.Binary":
                                                    return (
                                                        <>
                                                            <span className="done">workflow.Done</span>
                                                            <div className={"result"}>
                                                                <img
                                                                    src={`data:image/jpeg;base64,${done.Result["schema.Binary"]}`}
                                                                    alt=""/>
                                                            </div>
                                                            <WorkflowToString runID={done.BaseState?.RunID}/>
                                                            <ListVariables data={done.BaseState}/>
                                                        </>
                                                    )
                                                case "schema.String":
                                                    let funcName = "non"

                                                    if (done.BaseState?.Flow?.$type === "workflow.Flow") {
                                                        funcName = done.BaseState?.Flow["workflow.Flow"].Name!
                                                    }

                                                    let build = (type: string, funcName: string): predicate.Predicate => {
                                                        return {
                                                            "$type": "predicate.Compare",
                                                            "predicate.Compare": {
                                                                Location: `Data["${type}"].BaseState.Flow["workflow.Flow"].Name`,
                                                                Operation: "==",
                                                                BindValue: {
                                                                    "$type": "predicate.Literal",
                                                                    "predicate.Literal": {
                                                                        Value: {
                                                                            "$type": "schema.String",
                                                                            "schema.String": funcName
                                                                        }
                                                                    }
                                                                },
                                                            }
                                                        }
                                                    }

                                                    return <>
                                                        <span className="done">workflow.Done</span>
                                                        <button onClick={() => ctx.filter({
                                                            Predicate: {
                                                                "$type": "predicate.Or",
                                                                "predicate.Or": {
                                                                    L: [
                                                                        build("workflow.Done", funcName),
                                                                        build("workflow.Error", funcName),
                                                                        build("workflow.Await", funcName),
                                                                        build("workflow.Scheduled", funcName),
                                                                        build("workflow.ScheduleStopped", funcName),
                                                                    ]
                                                                }
                                                            }
                                                        })}>show {funcName}</button>
                                                        <button onClick={() => ctx.filter({
                                                            Predicate: {
                                                                "$type": "predicate.Compare",
                                                                "predicate.Compare": {
                                                                    Location: `Data["$type"]`,
                                                                    Operation: "==",
                                                                    BindValue: {
                                                                        "$type": "predicate.Literal",
                                                                        "predicate.Literal": {
                                                                            Value: {
                                                                                "$type": "schema.String",
                                                                                "schema.String": "workflow.Done"
                                                                            }
                                                                        }
                                                                    },
                                                                }
                                                            }
                                                        })}
                                                                className={"filter filter-in"}>show only
                                                        </button>
                                                        <button onClick={() => ctx.filter({
                                                            Predicate: {
                                                                "$type": "predicate.Compare",
                                                                "predicate.Compare": {
                                                                    Location: `Data["$type"]`,
                                                                    Operation: "!=",
                                                                    BindValue: {
                                                                        "$type": "predicate.Literal",
                                                                        "predicate.Literal": {
                                                                            Value: {
                                                                                "$type": "schema.String",
                                                                                "schema.String": "workflow.Done"
                                                                            }
                                                                        }
                                                                    },
                                                                }
                                                            }
                                                        })}
                                                                className={"filter filter-out"}>exclude
                                                        </button>
                                                        <div className={"result"}>
                                                            {done.Result["schema.String"]}
                                                        </div>

                                                        <WorkflowToString runID={done.BaseState?.RunID}/>
                                                        <ListVariables data={done.BaseState}/>
                                                    </>
                                            }

                                            return <>
                                                <span className="unknown">{done.Result?.$type}</span>
                                            </>

                                        case "workflow.Error":
                                            let error = data["workflow.Error"]
                                            return <>
                                                <span className="error">workflow.Error</span>
                                                <dl>
                                                    <dt>Code</dt>
                                                    <dd>{error.Code}</dd>
                                                    <dt>Message</dt>
                                                    <dd>{error.Reason}</dd>
                                                    <dt>Retried</dt>
                                                    <dd>{error.Retried} / {error.BaseState?.DefaultMaxRetries}</dd>
                                                </dl>
                                                <WorkflowToString runID={error.BaseState?.RunID}/>
                                                <ListVariables data={error.BaseState}/>
                                                <TryRecover error={error} onFinish={() => ctx.refresh()}/>
                                            </>

                                        case "workflow.Await":
                                            let await_ = data["workflow.Await"]

                                            return (
                                                <>
                                                    <span className="await">workflow.Await</span>
                                                    <button onClick={() => ctx.filter({
                                                        Predicate: {
                                                            "$type": "predicate.Compare",
                                                            "predicate.Compare": {
                                                                Location: `Data["$type"]`,
                                                                Operation: "==",
                                                                BindValue: {
                                                                    "$type": "predicate.Literal",
                                                                    "predicate.Literal": {
                                                                        Value: {
                                                                            "$type": "schema.String",
                                                                            "schema.String": "workflow.Await"
                                                                        }
                                                                    }
                                                                },
                                                            }
                                                        }
                                                    })}
                                                            className={"filter filter-in"}>only
                                                    </button>
                                                    <WorkflowToString runID={await_.BaseState?.RunID}/>
                                                    <ListVariables data={await_.BaseState}/>

                                                    <input type={"text"}
                                                           id="callbackValue"
                                                           placeholder={"callback result"}/>
                                                    <button onClick={(e) => {
                                                        e.preventDefault()
                                                        if (!await_.CallbackID) {
                                                            return
                                                        }

                                                        submitCallbackResult(await_.CallbackID, {
                                                            "schema.String": (document.getElementById("callbackValue") as HTMLInputElement).value
                                                        }, (data) => {
                                                            ctx.refresh()
                                                            setState(data)
                                                        })
                                                    }}>
                                                        Submit callback result
                                                    </button>
                                                </>
                                            )

                                        case "workflow.Scheduled":
                                            let scheduled = data["workflow.Scheduled"]

                                            let parentRunID = "no ParentRunID"
                                            let parentRunIDData = extractParentRunID(data)
                                            if (parentRunIDData !== undefined) {
                                                parentRunID = parentRunIDData
                                            }

                                            return (
                                                <>
                                                    <span className="schedguled">workflow.Scheduled</span>
                                                    <span>{JSON.stringify(scheduled.ExpectedRunTimestamp)}</span>
                                                    <span>{parentRunID}</span>
                                                    <WorkflowToString runID={scheduled.BaseState?.RunID}/>
                                                    <ListVariables data={scheduled.BaseState}/>
                                                    <button onClick={() => {
                                                        stopSchedule(parentRunID).finally(() => {
                                                            ctx.refresh()
                                                        })
                                                    }}>
                                                        Stop Schedule
                                                    </button>
                                                </>
                                            )

                                        case "workflow.ScheduleStopped":
                                            let scheduleStopped = data["workflow.ScheduleStopped"]

                                            let parentRunID1 = "no ParentRunID"
                                            let parentRunIDData1 = extractParentRunID(data)
                                            if (parentRunIDData1 !== undefined) {
                                                parentRunID1 = parentRunIDData1
                                            }

                                            return <>
                                                <span className="stopped">workflow.ScheduleStopped</span>
                                                <WorkflowToString runID={scheduleStopped.BaseState?.RunID}/>
                                                <ListVariables data={scheduleStopped.BaseState}/>
                                                <button onClick={() => {
                                                    resumeSchedule(parentRunID1).finally(() => {
                                                        ctx.refresh()
                                                    })
                                                }}>
                                                    Resume Schedule
                                                </button>
                                            </>
                                    }

                                    return <>
                                        <span className="unknown">{data.$type}</span>
                                    </>
                                }}/>
                        </td>
                    </tr>
                    </tbody>
                </table>
            </main>
        </div>
    )
        ;
}

export default App;

function ListVariables(props: { data?: workflow.BaseState }) {
    if (!props.data) {
        return <></>
    }

    return (
        <table>
            <tbody>
            {props.data?.Variables && Object.keys(props.data.Variables).length > 0 &&
                <>
                    <tr>
                        <td colSpan={2}>Variables</td>
                    </tr>
                    <tr>
                        <td>Key</td>
                        <td>Value</td>
                    </tr>
                </>
            }
            {props.data?.Variables && Object.keys(props.data.Variables).map((key) => {
                let val = props.data?.Variables?.[key]
                return (
                    <tr key={key}>
                        <td>{key}</td>
                        <td><SchemaValue data={val}/></td>
                    </tr>
                );
            })}
            {props.data?.ExprResult && Object.keys(props.data.ExprResult).length > 0 &&
                <>
                    <tr>
                        <td colSpan={2}>ExprResult</td>
                    </tr>
                    <tr>
                        <td>Key</td>
                        <td>Value</td>
                    </tr>
                </>
            }
            {props.data?.ExprResult && Object.keys(props.data.ExprResult).map((key) => {
                let val = props.data?.ExprResult?.[key]
                return (
                    <tr key={key}>
                        <td>{key}</td>
                        <td><SchemaValue data={val}/></td>
                    </tr>
                );
            })}
            </tbody>
        </table>
    );
}

function NativeValue(props: { data: any }) {
    switch (typeof props.data) {
        case "string":
            if (props.data.length > 500) {
                return <>{props.data.substring(0, 10)}...</>
            }
            return <>{props.data}</>
        case "number":
            return <>{props.data}</>
        case "boolean":
            return <>{props.data}</>
        case "object":
            if (props.data?.$type) {
                return <SchemaValue data={props.data}/>
            }

            return <table>
                <tbody>
                {Object.keys(props.data).map((key) => {
                    let val = props.data?.[key]
                    return (
                        <tr key={key}>
                            <td>{key}</td>
                            <td><SchemaValue data={val}/></td>
                        </tr>
                    );
                })}
                </tbody>
            </table>

    }

    return <>{JSON.stringify(props.data)}</>
}

function SchemaValue(props: { data?: schema.Schema }) {
    switch (props.data?.$type) {
        case "schema.String":
            return <>{props.data["schema.String"]}</>
        case "schema.Number":
            return <>{props.data["schema.Number"]}</>
        case "schema.Binary":
            return <>binary</>
        case "schema.Bool":
            return <>{props.data["schema.Bool"]}</>
        case "schema.List":
            const listData = props.data["schema.List"];
            return (
                <ul>
                    {listData && listData.map((item, index) => (
                        <li key={"list-item-" + index}>
                            <SchemaValue data={item}/>
                        </li>
                    ))}
                </ul>
            );

        case "schema.Map":
            const mapData = props.data["schema.Map"];
            const keys = Object.keys(mapData);

            if (keys && keys.length === 0) {
                return null; // If the map is empty, return null (no table to display)
            }

            return (
                <table>
                    <thead>
                    <tr>
                        <th>Key</th>
                        <th>Value</th>
                    </tr>
                    </thead>
                    <tbody>
                    {keys && keys.map((key) => (
                        <tr key={key}>
                            <td className="key">{key}</td>
                            <td>
                                <SchemaValue data={mapData[key]}/>
                            </td>
                        </tr>
                    ))}
                    </tbody>
                </table>
            );

    }

    return <NativeValue data={props.data}/>
}


function WorkflowToString(props: { flow?: workflow.Workflow, runID?: string}) {
    const [str, setStr] = useState("")

    useEffect(() => {
        if (props.flow) {
            flowToStringFromWorkflow(props.flow).then((data) => {
                setStr(data)
            })
        } else if (props.runID) {
            flowToStringFromRunID(props.runID).then((data) => {
                setStr(data)
            })
        }

    }, [props.flow, props.runID])

    return <>
        <pre>{str}</pre>
    </>
}

function SchedguledRun(props: { input: string }) {
    /*
    * flow book_product(input) {
    *    let reservation = BookReservation(input.productId, input.userId, input.quantity) @timeout(1m)
    *
    *    let user_payment_info, problem = await GetUserPaymentInfo() @timeout(5m) or input.user_payment_info
    *    if user_payment_info.err || problem.timeout {
    *      let canceled = CancelReservation(reservation)
    *      if canceled.err {
    *        return {err: "payment failed and reservation cancelation failed"}
    *      }
    *
    *      return {err: "payment failed, no use payment info"}
    *    }
    *
    *    let payment, problem = await ProcessPayment(user_payment_info) @timeout(24h)
    *    if payment.err || problem.timeout {
    *       let canceled = CancelReservation(reservation)
    *       if canceled.err {
    *           return {err: "payment failed and reservation cancelation failed"}
    *       }
    *
    *      return return {err: "payment failed, payment processing failed"}
    *    }
    *
    *    return return {ok: true, reservation, payment}
    * }
    */
    const cmd: workflow.Command = {
        "$type": "workflow.Run",
        "workflow.Run": {
            Flow: {
                "$type": "workflow.Flow",
                "workflow.Flow": {
                    Name: "create_attachment",
                    Arg: "input",
                    Body: [
                        {
                            "$type": "workflow.Choose",
                            "workflow.Choose": {
                                If: {
                                    "$type": "workflow.Compare",
                                    "workflow.Compare": {
                                        Operation: "=",
                                        Left: {
                                            "$type": "workflow.GetValue",
                                            "workflow.GetValue": {
                                                Path: "input",
                                            }
                                        },
                                        Right: {
                                            "$type": "workflow.SetValue",
                                            "workflow.SetValue": {
                                                Value: {
                                                    "schema.String": "666",
                                                },
                                            },
                                        },
                                    }
                                },
                                Then: [
                                    {
                                        "$type": "workflow.End",
                                        "workflow.End": {
                                            Result: {
                                                "$type": "workflow.SetValue",
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "Do no evil",
                                                    },
                                                }
                                            },
                                        },
                                    }
                                ],
                            }
                        },
                        {
                            "$type": "workflow.Assign",
                            "workflow.Assign": {
                                VarOk: "res",
                                Val: {
                                    "$type": "workflow.Apply",
                                    "workflow.Apply": {
                                        Name: "concat",
                                        Args: [
                                            {
                                                "$type": "workflow.SetValue",
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "hello ",
                                                    }
                                                }
                                            },
                                            {
                                                "$type": "workflow.GetValue",
                                                "workflow.GetValue": {
                                                    Path: "input",
                                                }
                                            },
                                        ]
                                    }
                                }
                            },
                        },
                        {
                            "$type": "workflow.End",
                            "workflow.End": {
                                Result: {
                                    "$type": "workflow.GetValue",
                                    "workflow.GetValue": {
                                        Path: "res",
                                    }
                                }
                            }
                        }
                    ],
                },
            },
            Input: {
                "schema.String": props.input,
            },
            RunOption: {
                "$type": "workflow.ScheduleRun",
                "workflow.ScheduleRun": {
                    Interval: "@every 10s"
                },
                // "workflow.DelayRun": {
                //     DelayBySeconds: 10,
                // },
            }
        }
    }

    const doIt = (e: React.MouseEvent) => {
        e.preventDefault()

        if (cmd?.["workflow.Run"]?.Flow) {
            flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
        }

        fetch('http://localhost:8080/', {
            method: 'POST',
            body: JSON.stringify(cmd),
        })
            .then(res => res.json())
        // .then(data => setState(data))
    }

    return <button onClick={doIt}>
        Scheduled Run
    </button>
}

const recover = (runID: string) => {
    const cmd: workflow.Command = {
        "$type": "workflow.TryRecover",
        "workflow.TryRecover": {
            RunID: runID,
        }
    }

    return fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => data as workflow.State)
}

function TryRecover(props: { error: workflow.Error, onFinish: (data: workflow.State) => void }) {
    return <button
        onClick={() => props.error.BaseState?.RunID && recover(props.error.BaseState?.RunID).then(props.onFinish)}>
        Try recover
    </button>
}

function stopSchedule(parentRunID: string) {
    const cmd: workflow.Command = {
        "$type": "workflow.StopSchedule",
        "workflow.StopSchedule": {
            ParentRunID: parentRunID,
        }
    }

    return fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => data as workflow.State)
}


function resumeSchedule(parentRunID: string) {
    const cmd: workflow.Command = {
        "$type": "workflow.ResumeSchedule",
        "workflow.ResumeSchedule": {
            ParentRunID: parentRunID,
        }
    }

    return fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => data as workflow.State)
}

function callFunc(funcID: string, args: any[]) {
    const cmd: workflow.FunctionInput = {
        Name: funcID,
        Args: args,
    }

    return fetch('http://localhost:8080/func', {
        method: 'POST',
        body: JSON.stringify(cmd),
    })
        .then(res => res.json())
        .then(data => data as workflow.FunctionOutput)
}


type HelloWorldDemoState = {
    input: string
    loading: boolean
}

type HelloWorldDemoAction = {
    type: string
}

function HelloWorldDemo() {
    const [state, setState] = React.useState({
        input: "Amigo",
        loading: false,
    } as HelloWorldDemoState);

    const dispatch = (action: HelloWorldDemoAction) => {
        switch (action.type) {
            case "run_hello_world":
                setState({
                    ...state,
                    loading: true,
                })

                runHelloWorldWorkflow(state.input, (data) => {
                    setState({
                        ...state,
                        loading: false,
                    })
                })
                break

            case "run_hello_world_error":
                setState({
                    ...state,
                    loading: true,
                })

                runErrorWorkflow(state.input, (data) => {
                    setState({
                        ...state,
                        loading: false,
                    })
                })
                break
        }
    }


    return <form
        className={"action-section"}>
        <h2>Hello world demo</h2>
        <input type="text"
               placeholder="Enter your name"
               value={state.input}
               onChange={(e) => {
                   setState({
                       ...state,
                       input: e.currentTarget.value,
                   })
               }}
        />
        <button onClick={() => dispatch({type: "run_hello_world"})}>
            Run hello world workflow
        </button>
        <button onClick={() => dispatch({type: "run_hello_world_error"})}>
            Run hello world workflow with error
        </button>
        {state.loading && <div>Loading...</div>}
    </form>
}

