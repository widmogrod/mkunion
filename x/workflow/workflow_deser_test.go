package workflow

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	val := `{
  "$type": "workflow.Flow",
  "workflow.Flow": {
    "Name": "hello_world",
    "Arg": "input",
    "Body": [
      {
        "$type": "workflow.Choose",
        "workflow.Choose": {
          "ID": "choose1",
          "If": {
            "$type": "workflow.Compare",
            "workflow.Compare": {
              "Operation": "=",
              "Left": {
                "$type": "workflow.GetValue",
                "workflow.GetValue": {
                  "Path": "input"
                }
              },
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
          ],
          "Else": null
        }
      },
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
              "Name": "concat",
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
              "Await": null
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
              "Path": "res"
            }
          }
        }
      }
    ]
  }
}`

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
      "schema.Map": {
		"prompt": {"schema.String": "hello world"},
		"width": {"schema.Number": 100},
		"height": {"schema.Number": 100}
      }
    }
  }
}`

	res, err := CommandFromJSON([]byte(val))
	assert.NoError(t, err)

	out, err := CommandToJSON(res)
	assert.NoError(t, err)

	t.Log(string(out))

}
