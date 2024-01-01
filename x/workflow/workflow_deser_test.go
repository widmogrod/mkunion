package workflow

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	val := `{
  "$type": "workflow.Flow",
  "workflow.Flow": {
    "Arg": "input",
    "Body": [
      {
        "$type": "workflow.Choose",
        "workflow.Choose": {
          "Else": [],
          "ID": "choose1",
          "If": {
            "$type": "workflow.Compare",
            "workflow.Compare": {
              "Left": {
                "$type": "workflow.GetValue",
                "workflow.GetValue": {
                  "Path": "input"
                }
              },
              "Operation": "=",
              "Right": {
                "$type": "workflow.SetValue",
                "workflow.SetValue": {
                  "Value": {
                    "$type": "schema.String",
                    "schema.String": "666"
                  }
                }
              }
            }
          },
          "Then": [
            {
              "$type": "workflow.End",
              "workflow.End": {
                "ID": "end2",
                "Result": {
                  "$type": "workflow.SetValue",
                  "workflow.SetValue": {
                    "Value": {
                      "$type": "schema.String",
                      "schema.String": "Do no evil"
                    }
                  }
                }
              }
            }
          ]
        }
      },
      {
        "$type": "workflow.Assign",
        "workflow.Assign": {
          "ID": "assign1",
          "Val": {
            "$type": "workflow.Apply",
            "workflow.Apply": {
              "Args": [
                {
                  "$type": "workflow.SetValue",
                  "workflow.SetValue": {
                    "Value": {
                      "$type": "schema.String",
                      "schema.String": "hello "
                    }
                  }
                },
                {
                  "$type": "workflow.GetValue",
                  "workflow.GetValue": {
                    "Path": "input"
                  }
                }
              ],
              "ID": "apply1",
              "Name": "concat"
            }
          },
          "VarErr": "",
          "VarOk": "res"
        }
      },
      {
        "$type": "workflow.End",
        "workflow.End": {
          "ID": "end1",
          "Result": {
            "$type": "workflow.GetValue",
            "workflow.GetValue": {
              "Path": "res"
            }
          }
        }
      }
    ],
    "Name": "hello_world"
  }
}
`

	res, err := WorflowFromJSON([]byte(val))
	assert.NoError(t, err)

	out, err := WorflowToJSON(res)
	assert.NoError(t, err)
	t.Log(string(out))

	assert.JSONEq(t, val, string(out))
}

func TestSecond(t *testing.T) {
	val := `{
  "$type": "workflow.Run",
  "workflow.Run": {
    "Flow": {
      "$type": "workflow.Flow",
      "workflow.Flow": {
        "Name": "generateandresizeimage",
        "Arg": "input",
        "Body": [
          {
            "$type": "workflow.Assign",
            "workflow.Assign": {
              "ID": "assign1",
              "VarOk": "res",
              "VarErr": "",
              "Val": {
                "$type": "workflow.Apply",
                "workflow.Apply": {
                  "ID": "apply1",
                  "Name": "genimageb64",
                  "Args": [
                    {
                      "$type": "workflow.GetValue",
                      "workflow.GetValue": {
                        "Path": "input.prompt"
                      }
                    }
                  ]
                }
              }
            }
          },
          {
            "$type": "workflow.Assign",
            "workflow.Assign": {
              "ID": "assign2",
              "VarOk": "res_small",
              "VarErr": "",
              "Val": {
                "$type": "workflow.Apply",
                "workflow.Apply": {
                  "ID": "apply2",
                  "Name": "resizeimgb64",
                  "Args": [
                    {
                      "$type": "workflow.GetValue",
                      "workflow.GetValue": {
                        "Path": "res"
                      }
                    },
                    {
                      "$type": "workflow.GetValue",
                      "workflow.GetValue": {
                        "Path": "input.width"
                      }
                    },
                    {
                      "$type": "workflow.GetValue",
                      "workflow.GetValue": {
                        "Path": "input.height"
                      }
                    }
                  ]
                }
              }
            }
          },
          {
            "$type": "workflow.End",
            "workflow.End": {
              "ID": "end1",
              "Result": {
                "$type": "workflow.GetValue",
                "workflow.GetValue": {
                  "Path": "res_small"
                }
              }
            }
          }
        ]
      }
    },
    "Input": {
     "$type": "schema.Map",
      "schema.Map": {
        "height": {
          "$type": "schema.Number",
          "schema.Number": 100
        },
        "prompt": {
          "$type": "schema.String",
          "schema.String": "hello world"
        },
        "width": {
          "$type": "schema.Number",
          "schema.Number": 100
        }
      }
    },
    "RunOption": null
  }
}`

	res, err := CommandFromJSON([]byte(val))
	assert.NoError(t, err)

	out, err := CommandToJSON(res)
	assert.NoError(t, err)

	t.Log(string(out))

	assert.JSONEq(t, val, string(out))
}
