import React, {useEffect, useReducer, useState} from 'react';
import './App.css';
import * as openai from './workflow/'
import * as schemaless from './workflow/github_com_widmogrod_mkunion_x_storage_schemaless'
import * as workflow from './workflow/github_com_widmogrod_mkunion_x_workflow'
import * as schema from "./workflow/github_com_widmogrod_mkunion_x_schema";
import {Chat} from "./Chat";

function flowCreate(flow: workflow.Flow) {
    return fetch('http://localhost:8080/flow', {
        method: 'POST',
        body: JSON.stringify(flow),
    })
        .then(res => res.text())
        .then(data => console.log("save-flow-result", data))
}

function flowToString(flow: workflow.Worflow) {
    return fetch('http://localhost:8080/workflow-to-str', {
        method: 'POST',
        body: JSON.stringify(flow),
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
            })
        } as schemaless.FindingRecords<schemaless.Record<T>>),
    })
        .then(res => res.json())
        .then(data => data as schemaless.PageResult<schemaless.Record<T>>)
}

function listStates(input?: ListProps<workflow.State>) {
    return storageList<workflow.State>({
        path: "states",
        sort: input?.sort,
        limit: input?.limit,
    })
}

function listFlows(input?: ListProps<workflow.Flow>) {
    return storageList<workflow.Flow>({
        path: "flows",
        sort: input?.sort,
        limit: input?.limit,
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
                                            Timeout: 10,
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

                <form className={"action-section"}>
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
                </form>
            </aside>
            <main>
                <table>
                    <tbody>
                    <tr>
                        <td>
                            <PaginatedTable<workflow.Flow>
                                load={(state) => {
                                    return listFlows({
                                        limit: state.limit,
                                        sort: state.sort,
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
                        </td>
                        <td>
                            <PaginatedTable<workflow.State>
                                load={(state) => {
                                    return listStates({
                                        limit: state.limit,
                                        sort: state.sort,
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
                                mapData={(input) => {
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
                                                            <img
                                                                src={`data:image/jpeg;base64,${done.Result["schema.Binary"]}`}
                                                                alt=""/>
                                                            <ListVariables data={done.BaseState}/>
                                                        </>
                                                    )
                                                case "schema.String":
                                                    return <>
                                                        <span className="done">workflow.Done</span>
                                                        {done.Result["schema.String"]}
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
                                                </dl>
                                                <ListVariables data={error.BaseState}/>
                                                <TryRecover error={error}/>
                                            </>

                                        case "workflow.Await":
                                            let await_ = data["workflow.Await"]

                                            return (
                                                <>
                                                    <span className="await">workflow.Await</span>
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
                                                    <ListVariables data={scheduled.BaseState}/>
                                                    <button onClick={() => {
                                                        stopSchedule(parentRunID)
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
                                                <ListVariables data={scheduleStopped.BaseState}/>
                                                <button onClick={() => {
                                                    resumeSchedule(parentRunID1)
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
                        <td>
                            <img src={`data:image/jpeg;base64,${image}`} alt=""/>
                            <pre>Func output: {output}</pre>
                            <pre>Workflow output: {JSON.stringify(state, null, 2)} </pre>
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

type PaginatedTableState<T> = {
    limit: number
    sort: PaginatedTableSort
    selected: { [key: string]: schemaless.Record<T> }
}

type PaginatedTableSort = {
    [key: string]: boolean
}

type PaginatedTableAction<T> = {
    name: string
    action: (state: PaginatedTableState<T>, ctx: PaginatedTableContext<T>) => void
}

type PaginatedTableContext<T> = {
    refresh: () => void
    clearSelection: () => void
}


type PaginatedTableProps<T> = {
    limit?: number
    sort?: PaginatedTableSort
    load: (input: PaginatedTableState<T>) => Promise<schemaless.PageResult<schemaless.Record<T>>>
    mapData?: (data: schemaless.Record<T>) => JSX.Element
    actions?: PaginatedTableAction<T>[]
}

function PaginatedTable<T>(props: PaginatedTableProps<T>) {
    const [data, setData] = useState({Items: [] as any[]} as schemaless.PageResult<schemaless.Record<T>>)
    const [state, setState] = useState({
        limit: props.limit || 30,
        sort: props.sort || {},
        selected: {},
    } as PaginatedTableState<T>)

    useEffect(() => {
        props.load(state).then(setData)
    }, [state])

    const ctx = {
        refresh: () => {
            props.load(state).then(setData)
        },
        clearSelection: () => {
            setState({
                ...state,
                selected: {},
            })
        }
    } as PaginatedTableContext<T>

    const changeSort = (key: string) => (e: React.MouseEvent) => {
        e.preventDefault()

        let newSort = {...state.sort}
        if (newSort[key] === undefined) {
            newSort[key] = true
        } else if (newSort[key]) {
            newSort[key] = false
        } else {
            delete newSort[key]
        }

        setState({
            ...state,
            sort: newSort,
        })
    }

    const sortState = (key: string) => {
        if (state.sort[key] === undefined) {
            return "sort-none"
        } else if (state.sort[key]) {
            return "sort-asc"
        } else {
            return "sort-desc"
        }
    }


    const selectRowToggle = (item: schemaless.Record<T>) => () => {
        if (!item.ID) {
            return
        }

        let selected = {...state.selected}
        if (selected[item.ID]) {
            delete selected[item.ID]
        } else {
            selected[item.ID] = item
        }

        setState({
            ...state,
            selected: selected,
        })
    }

    const isSelected = (item: schemaless.Record<T>) => {
        if (!item.ID) {
            return false
        }

        return state.selected[item.ID] !== undefined
    }

    const batchSelection = (e: React.MouseEvent) => {
        e.preventDefault()

        let selectionLength = Object.keys(state.selected).length
        if (selectionLength > 0) {
            setState({
                ...state,
                selected: {},
            })
        } else {
            let selected = {} as { [key: string]: schemaless.Record<T> }
            data.Items?.forEach((item) => {
                if (!item.ID) {
                    return
                }

                selected[item.ID] = item
            })

            setState({
                ...state,
                selected: selected
            } as PaginatedTableState<T>)
        }
    }

    const batchSelectionState = () => {
        let selectionLength = Object.keys(state.selected).length
        if (selectionLength === 0) {
            return "selected-none"
        }
        if (selectionLength === data.Items?.length) {
            return "selected-all"
        }

        return "selected-some"
    }

    const applyAction = (action: PaginatedTableAction<T>) => (e: React.MouseEvent) => {
        e.preventDefault()
        action.action(state, ctx)
    }

    return <table>
        <thead>
        <tr>
            <th colSpan={5} className={"option-row"}>
                <button className={"refresh"} onClick={() => ctx.refresh()}>Refresh</button>

                {props.actions && props.actions.map((action) => {
                    return <button
                        key={action.name}
                        className={"action "}
                        disabled={Object.keys(state.selected).length === 0}
                        onClick={applyAction(action)}>{action.name}</button>
                })}
            </th>
        </tr>
        <tr>
            <th onClick={batchSelection} className={batchSelectionState()}>
                <button>select</button>
            </th>
            <th onClick={changeSort("ID")} className={sortState("ID")}>ID</th>
            <th onClick={changeSort("Type")} className={sortState("Type")}>Type</th>
            <th onClick={changeSort("Version")} className={sortState("Version")}>Version</th>
            <th>Data</th>
        </tr>
        </thead>
        <tbody>
        {data.Items && data.Items.length > 0 ? data.Items.map((item) => {
            return (
                <tr key={item.ID}>
                    <td><input type={"checkbox"} onChange={selectRowToggle(item)} checked={isSelected(item)}/></td>
                    <td>{item.ID}</td>
                    <td>{item.Type}</td>
                    <td>{item.Version}</td>
                    <td>{props.mapData && props.mapData(item)}</td>
                </tr>
            );
        }) : (
            <tr>
                <td colSpan={5}>No data</td>
            </tr>
        )}
        </tbody>
    </table>
}

function WorkflowToString(props: { flow: workflow.Worflow }) {
    const [str, setStr] = useState("")

    useEffect(() => {
        flowToString(props.flow).then((data) => {
            setStr(data)
        })
    }, [props.flow])

    return <>
        {/*<pre>{JSON.stringify(props.flow)}</pre>*/}
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

function TryRecover(props: { error: workflow.Error }) {
    return <button onClick={() => props.error.BaseState?.RunID && recover(props.error.BaseState?.RunID)}>
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
        Name: "funcID",
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


    return <div
        className={"action-section"}>
        <h2>Hello world demo</h2>
        <input type="text"
               placeholder="Enter your name"
               value={state.input}
        />
        <button onClick={() => dispatch({type: "run_hello_world"})}>
            Run hello world workflow
        </button>
        <button onClick={() => dispatch({type: "run_hello_world_error"})}>
            Run hello world workflow with error
        </button>
        {state.loading && <div>Loading...</div>}
    </div>
}

