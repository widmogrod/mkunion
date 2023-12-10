import React, {useEffect, useState} from 'react';
import './App.css';
import * as workflow from './workflow/workflow'
import {dediscriminateCommand} from './workflow/workflow'
import * as schema from "./workflow/github_com_widmogrod_mkunion_x_schema";
import {Chat} from "./Chat";
import {GenerateImage, ListWorkflowsFn} from "./workflow/main";

function flowCreate(flow: workflow.Flow) {
    console.log("save-flow", flow)
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

type record = {
    ID: string,
    Type: string,
    Data: workflow.State
}

function listStates(onData: (data: { Items: record[] }) => void) {
    fetch('http://localhost:8080/list', {
        method: 'GET',
    })
        .then(res => res.json())
        .then(data => {
            onData(data);
        })
}

type recordFlow = {
    ID: string,
    Type: string,
    Data: workflow.Flow
}

function listFlows(onData: (data: { Items: recordFlow[] }) => void) {
    fetch('http://localhost:8080/flows', {
        method: 'GET',
    })
        .then(res => res.json())
        .then(data => {
            onData(data);
        })
}

function runFlow(flowID: string, input: string, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "workflow.Run": {
            Flow: {
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
        body: JSON.stringify(dediscriminateCommand(cmd)),
    })
        .then(res => res.json())
        .then(data => {
            onData && onData(data)
        })

}

function runHelloWorldWorkflow(input: string, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "workflow.Run": {
            Flow: {
                "workflow.Flow": {
                    Name: "hello_world",
                    Arg: "input",
                    Body: [
                        {
                            "workflow.Choose": {
                                ID: "choose1",
                                If: {
                                    "workflow.Compare": {
                                        Operation: "=",
                                        Left: {
                                            "workflow.GetValue": {
                                                Path: "input",
                                            }
                                        },
                                        Right: {
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
                                        "workflow.End": {
                                            ID: "end2",
                                            Result: {
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
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                Val: {
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "concat",
                                        Args: [
                                            {
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "hello ",
                                                    }
                                                }
                                            },
                                            {
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
                            "workflow.End": {
                                ID: "end1",
                                Result: {
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
        body: JSON.stringify(dediscriminateCommand(cmd)),
    })
        .then(res => res.json())
        .then(data => onData && onData(data))
}

function generateImage(imageWidth: number, imageHeight: number, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "workflow.Run": {
            Flow: {
                "workflow.Flow": {
                    Name: "generateandresizeimage",
                    Arg: "input",
                    Body: [
                        {
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                Val: {
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "genimageb64",
                                        Args: [
                                            {
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
                            "workflow.Assign": {
                                ID: "assign2",
                                VarOk: "res_small",
                                Val: {
                                    "workflow.Apply": {
                                        ID: "apply2",
                                        Name: "resizeimgb64",
                                        Args: [
                                            {
                                                "workflow.GetValue": {
                                                    Path: "res",
                                                }
                                            },
                                            {
                                                "workflow.GetValue": {
                                                    Path: "input.width",
                                                }
                                            },
                                            {
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
                            "workflow.End": {
                                ID: "end1",
                                Result: {
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
                    "prompt": "no text",
                    "width": imageWidth,
                    "height": imageHeight,
                },
            },
        }
    }

    if (cmd?.["workflow.Run"]?.Flow) {
        flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
    }

    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(dediscriminateCommand(cmd)),
    })
        .then(res => res.json())
        .then((data: workflow.State) => {
            onData && onData(data)

        })
}

function runContactAwait(imageWidth: number, imageHeight: number, onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "workflow.Run": {
            Flow: {
                "workflow.Flow": {
                    Name: "concat_await",
                    Arg: "input",
                    Body: [
                        {
                            "workflow.Assign": {
                                ID: "assign1",
                                VarOk: "res",
                                Val: {
                                    "workflow.Apply": {
                                        ID: "apply1",
                                        Name: "concat",
                                        Args: [
                                            {
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "await hello ",
                                                    }
                                                }
                                            },
                                            {
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
                            "workflow.End": {
                                ID: "end1",
                                Result: {
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
                    "prompt": "no text",
                    "width": imageWidth,
                    "height": imageHeight,
                },
            },
        }
    }

    if (cmd?.["workflow.Run"]?.Flow) {
        flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
    }

    fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(dediscriminateCommand(cmd)),
    })
        .then(res => res.json())
        .then((data: workflow.State) => {
            onData && onData(data)
        })
}

function submitCallbackResult(onData?: (data: workflow.State) => void) {
    const cmd: workflow.Command = {
        "workflow.Callback": {
            CallbackID: "callback_id",
            Result: {
                "schema.String": "callback result",
            },
        }
    }

    fetch('http://localhost:8080/callback', {
        method: 'POST',
        body: JSON.stringify(dediscriminateCommand(cmd)),
    })
        .then(res => res.json())
        .then((data: workflow.State) => {
            onData && onData(data)
        })
}

function App() {
    const [state, setState] = React.useState({} as workflow.State);
    const [input, setInput] = React.useState("hello");
    const [output, setOutput] = React.useState("" as any);


    const [table, setTable] = React.useState({Items: [] as record[]});

    const [image, setImage] = React.useState("" as string);
    const [imageWidth, setImageWidth] = React.useState(100 as number);
    const [imageHeight, setImageHeight] = React.useState(100 as number);
    const [selectedFlow, setSelectedFlow] = React.useState("hello_world" as string);


    const [flows, setFlows] = React.useState({Items: [] as recordFlow[]});


    return (
        <div className="App">
            <main>
                <h1>My App</h1>
                <form
                    className={"action-section"}
                    onSubmit={(e) => {
                        e.preventDefault()
                        runHelloWorldWorkflow(input)
                    }}
                >
                    <h2>Hello world</h2>
                    <input type="text"
                           placeholder="Enter your name"
                           onInput={(e) => setInput(e.currentTarget.value)}/>
                    <button>
                        Run hello world workflow
                    </button>
                </form>

                <form
                    className={"action-section"}
                    onSubmit={(e) => {
                        e.preventDefault()
                        generateImage(imageWidth, imageHeight, (data) => {
                            if ("workflow.Done" in data) {
                                setImage(data["workflow.Done"].Result["schema.Binary"]);
                            } else if ("workflow.Error" in data) {
                                console.log(data["workflow.Error"])
                            }
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

                <form className={"action-section"}>
                    <h2>Display tables</h2>
                    <button onClick={(e) => {
                        e.preventDefault()
                        listStates((data) => {
                            setTable(data);
                        })
                    }}>
                        List states
                    </button>

                    <button onClick={(e) => {
                        e.preventDefault()
                        listFlows((data) => {
                            setFlows(data);
                        })
                    }}>
                        List flows
                    </button>
                </form>

                <form
                    className={"action-section"}
                    onSubmit={(e) => {
                        e.preventDefault()
                        runFlow(selectedFlow, input, (data) => {
                            if ("workflow.Done" in data) {
                                setImage(data["workflow.Done"].Result["schema.Binary"]);
                            } else if ("workflow.Error" in data) {
                                console.log(data["workflow.Error"])
                            }
                        })
                    }}
                >
                    <h2>Run selected flow</h2>
                    <select value={selectedFlow}
                            onChange={(e) => setSelectedFlow(e.currentTarget.value)}>
                        {flows.Items.map((item) => {
                            return (
                                <option key={item.ID} value={item.ID}>{item.ID}</option>
                            );
                        })}
                    </select>

                    <button>
                        Run selected flow
                    </button>
                </form>

                <form className={"action-section"}>
                    <h2>Async and callback result</h2>
                    <button onClick={(e) => {
                        e.preventDefault()
                        runContactAwait(imageWidth, imageHeight, (data) => {
                            if ("workflow.Done" in data) {
                                setImage(data["workflow.Done"].Result["schema.Binary"]);
                            } else if ("workflow.Error" in data) {
                                console.log(data["workflow.Error"])
                            }
                        })
                    }
                    }>
                        Await image
                    </button>

                    <button onClick={(e) => {
                        e.preventDefault()
                        submitCallbackResult((data) => {
                            if ("workflow.Done" in data) {
                                setImage(data["workflow.Done"].Result["schema.Binary"]);
                            } else if ("workflow.Error" in data) {
                                console.log(data["workflow.Error"])
                            }
                        })
                    }}>
                        Submit callback result
                    </button>
                </form>

                <form className={"action-section"}>
                    <h2>Schedule run</h2>
                    <SchedguledRun input={input}/>
                </form>

                <form className={"action-section"}>
                    <h2>Invoke function without workflow</h2>
                    <button onClick={() => {
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

                <table>
                    <tbody>
                    <tr>
                        <td>
                            <Chat
                                props={{
                                    name: "John",
                                    onFunctionCall: (x: { Name: string, Arguments: string }) => {
                                        console.log("onFunctionCall", x);
                                        switch (x.Name) {
                                            case "count_words":
                                                let args = JSON.parse(x.Arguments) as ListWorkflowsFn
                                                console.log(args)
                                                break

                                            case "refresh_states":
                                                listStates(setTable)
                                                break;

                                            case "refresh_flows":
                                                listFlows(setFlows)
                                                break;

                                            case "generate_image":
                                                let args2 = JSON.parse(x.Arguments) as GenerateImage;
                                                generateImage(args2?.Width || 100, args2?.Height || 100, (data) => {
                                                    if ("workflow.Done" in data) {
                                                        setImage(data["workflow.Done"].Result["schema.Binary"]);
                                                    } else if ("workflow.Error" in data) {
                                                        console.log(data["workflow.Error"])
                                                    }
                                                    listStates(setTable)
                                                    listFlows(setFlows)
                                                })
                                                break;
                                        }
                                    }
                                }}
                            />
                        </td>
                        <td>
                            <PaginatedTable table={flows} mapData={(data: workflow.Flow) => {
                                return <WorkflowToString flow={{
                                    "workflow.Flow": data,
                                }}/>
                                // return <SchemaValue data={data}/>
                            }}/>
                        </td>
                        <td>
                            <PaginatedTable table={table} mapData={(data) => {
                                if ("workflow.Done" in data) {
                                    if ("schema.Binary" in data["workflow.Done"].Result) {
                                        return (
                                            <>
                                                <span className="done">workflow.Done</span>
                                                <img
                                                    src={`data:image/jpeg;base64,${data["workflow.Done"].Result["schema.Binary"]}`}
                                                    alt=""/>
                                                <ListVariables data={data["workflow.Done"].BaseState}/>
                                            </>
                                        )
                                    } else if ("schema.String" in data["workflow.Done"].Result) {
                                        return <>
                                            <span className="done">workflow.Done</span>
                                            {data["workflow.Done"].Result["schema.String"]}
                                            <ListVariables data={data["workflow.Done"].BaseState}/>
                                        </>
                                    }

                                    return JSON.stringify(data["workflow.Done"].Result)
                                } else if ("workflow.Error" in data) {
                                    return <>
                                        <span className="error">workflow.Error</span>
                                        {JSON.stringify(data["workflow.Error"])}
                                    </>
                                } else if ("workflow.Await" in data) {
                                    return (
                                        <>
                                            <span className="await">workflow.Await</span>
                                            <ListVariables data={data["workflow.Await"].BaseState}/>
                                        </>
                                    )
                                } else if ("workflow.Scheduled" in data) {
                                    return (
                                        <>
                                            <span className="schedguled">workflow.Scheduled</span>
                                            <span>{JSON.stringify(data["workflow.Scheduled"].ExpectedRunTimestamp)}</span>
                                            <ListVariables data={data["workflow.Scheduled"].BaseState}/>
                                            <button onClick={() => {
                                                stopSchedule(data["workflow.Scheduled"].ParentRunID)
                                            }}>
                                                Stop Schedule
                                            </button>
                                        </>
                                    )
                                } else if ("workflow.ScheduleStopped") {
                                    return <>
                                        <span className="stopped">workflow.ScheduleStopped</span>
                                        <ListVariables data={data["workflow.ScheduleStopped"].BaseState}/>
                                        <button onClick={() => {
                                            resumeSchedule(data["workflow.ScheduleStopped"].ParentRunID)
                                        }}>
                                            Resume Schedule
                                        </button>
                                    </>
                                } else {
                                    return JSON.stringify(data)
                                }
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
    );
}

export default App;

function ListVariables(props: { data: workflow.BaseState }) {
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
                let val = props.data.Variables?.[key]
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
                let val = props.data.ExprResult?.[key]
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

function SchemaValue(props: { data: schema.Schema }) {
    // check if props.data is an object
    if (typeof props.data !== 'object') {
        return <>{JSON.stringify(props.data)}</>
    }

    if ("schema.String" in props.data) {
        return <>{props.data["schema.String"]}</>
    } else if ("schema.Binary" in props.data) {
        return <>binary</>
    } else if ("schema.Map" in props.data) {
        const mapData = props.data["schema.Map"];
        const keys = Object.keys(mapData);

        if (keys.length === 0) {
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
                {keys.map((key) => (
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

    return <>{JSON.stringify(props.data)}</>
}

function PaginatedTable(props: { table: { Items: any[] }, mapData: (data: any) => any }) {
    const mapData = props.mapData || ((data: any) => JSON.stringify(data))
    return <table>
        <thead>
        <tr>
            <th>ID</th>
            <th>Type</th>
            <th>Version</th>
            <th>Data</th>
        </tr>
        </thead>
        <tbody>
        {props.table && props.table.Items && props.table.Items.map((item) => {
            return (
                <tr key={item.ID}>
                    <td>{item.ID}</td>
                    <td>{item.Type}</td>
                    <td>{item.Version}</td>
                    <td>{mapData(item.Data)}</td>
                </tr>
            );
        })}
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
        "workflow.Run": {
            Flow: {
                "workflow.Flow": {
                    Name: "create_attachment",
                    Arg: "input",
                    Body: [
                        {
                            "workflow.Choose": {
                                If: {
                                    "workflow.Compare": {
                                        Operation: "=",
                                        Left: {
                                            "workflow.GetValue": {
                                                Path: "input",
                                            }
                                        },
                                        Right: {
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
                                        "workflow.End": {
                                            Result: {
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
                            "workflow.Assign": {
                                VarOk: "res",
                                Val: {
                                    "workflow.Apply": {
                                        Name: "concat",
                                        Args: [
                                            {
                                                "workflow.SetValue": {
                                                    Value: {
                                                        "schema.String": "hello ",
                                                    }
                                                }
                                            },
                                            {
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
                            "workflow.End": {
                                Result: {
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
                "workflow.ScheduleRun": {
                    Interval: "@every 10s"
                },
                // "workflow.DelayRun": {
                //     DelayBySeconds: 10,
                // },
            }
        }
    }

    const doIt = () => {
        if (cmd?.["workflow.Run"]?.Flow) {
            flowCreate(cmd?.["workflow.Run"]?.Flow as workflow.Flow)
        }

        fetch('http://localhost:8080/', {
            method: 'POST',
            body: JSON.stringify(dediscriminateCommand(cmd)),
        })
            .then(res => res.json())
        // .then(data => setState(data))
    }

    return <button onClick={doIt}>
        Scheduled Run
    </button>

}

function stopSchedule(parentRunID: string) {
    const cmd: workflow.Command = {
        "workflow.StopSchedule": {
            ParentRunID: parentRunID,
        }
    }

    return fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(dediscriminateCommand(cmd)),
    })
        .then(res => res.json())
        .then(data => data as workflow.State)
}


function resumeSchedule(parentRunID: string) {
    const cmd: workflow.Command = {
        "workflow.ResumeSchedule": {
            ParentRunID: parentRunID,
        }
    }

    return fetch('http://localhost:8080/', {
        method: 'POST',
        body: JSON.stringify(dediscriminateCommand(cmd)),
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
