import React from 'react';
import './App.css';
import * as workflow from './workflow/workflow'
import {dediscriminateCommand} from './workflow/workflow'

function App() {
    const [state, setState] = React.useState({} as workflow.State);
    const [input, setInput] = React.useState("hello");

    type record = {
        ID: string,
        Type: string,
        Data: workflow.State
    }
    const [table, setTable] = React.useState({Items: [] as record[]});

    const [image, setImage] = React.useState("" as string);
    const [imageWidth, setImageWidth] = React.useState(100 as number);
    const [imageHeight, setImageHeight] = React.useState(100 as number);

    return (
        <div className="App">
            <main>
                <h1>My App</h1>
                <input type="text"
                       placeholder="Enter your name"
                       onInput={(e) => setInput(e.currentTarget.value)}/>
                <button onClick={() => {
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

                    fetch('http://localhost:8080/', {
                        method: 'POST',
                        body: JSON.stringify(dediscriminateCommand(cmd)),
                    })
                        .then(res => res.json())
                        .then(data => setState(data))
                }
                }>Run workflow
                </button>


                <input type="number"
                       placeholder="Width"
                       onInput={(e) => setImageWidth(parseInt(e.currentTarget.value))}/>
                <input type="number"
                       placeholder="Height"
                       onInput={(e) => setImageHeight(parseInt(e.currentTarget.value))}/>
                <button onClick={() => {
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

                    fetch('http://localhost:8080/', {
                        method: 'POST',
                        body: JSON.stringify(dediscriminateCommand(cmd)),
                    })
                        .then(res => res.json())
                        .then((data: workflow.State) => {
                            if ("workflow.Done" in data) {
                                setImage(data["workflow.Done"].Result["schema.String"])
                            } else if ("workflow.Error" in data) {
                                console.log(data["workflow.Error"])
                            }
                        })
                }
                }>Generate image
                </button>


                <button onClick={() => {
                    fetch('http://localhost:8080/list', {
                        method: 'GET',
                    })
                        .then(res => res.json())
                        .then(data => {
                            setTable(data);
                        })
                }}>
                    List states
                </button>

                <pre>{JSON.stringify(state, null, 2)} </pre>

                <img src={`data:image/jpeg;base64,${image}`} alt=""/>

                <table width="500px">
                    <thead>
                    <tr>
                        <th>ID</th>
                        <th>Type</th>
                        <th>State</th>
                        <th>Result</th>
                    </tr>
                    </thead>
                    <tbody>
                    {table && table.Items && table.Items.map((item) => {
                        let renderData;
                        if ("workflow.Done" in item.Data) {
                            renderData = (
                                <>
                                    <td>workflow.Done</td>
                                    <td>{item.Data["workflow.Done"].Result["schema.String"]}</td>
                                </>
                            );
                        } else if ("workflow.Error" in item.Data) {
                            renderData = (
                                <>
                                    <td>workflow.Error</td>
                                    <td>{item.Data["workflow.Error"].Code}</td>
                                    <td>{item.Data["workflow.Error"].Reason}</td>
                                </>
                            );
                        } else {
                            renderData = <td colSpan={4}>{JSON.stringify(item.Data)}</td>;
                        }
                        // const state = discriminateState(item.Data)
                        // switch (state.$type) {
                        //     case 'workflow.Done':
                        //         renderData = (
                        //             <>
                        //                 <td>{state.$type}</td>
                        //                 <td>{state["workflow.Done"].Result["schema.String"]}</td>
                        //             </>
                        //         );
                        //         break;
                        //     case 'workflow.Error':
                        //         renderData = (
                        //             <>
                        //                 <td>{state.$type}</td>
                        //                 <td>{state["workflow.Error"].Code}</td>
                        //                 <td>{state["workflow.Error"].Reason}</td>
                        //             </>
                        //         );
                        //         break;
                        //     default:
                        //         renderData = <td colSpan={4}>{JSON.stringify(item.Data)}</td>;
                        // }

                        return (
                            <tr key={item.ID}>
                                <td>{item.ID}</td>
                                <td>{item.Type}</td>
                                {renderData}
                            </tr>
                        );
                    })}
                    </tbody>
                </table>


            </main>
        </div>
    )
        ;
}

export default App;
