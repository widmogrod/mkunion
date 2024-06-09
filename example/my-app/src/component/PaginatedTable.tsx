import React, {useEffect, useState} from "react";
import * as schemaless from "../workflow/github_com_widmogrod_mkunion_x_storage_schemaless";
import * as schema from "../workflow/github_com_widmogrod_mkunion_x_schema";
import * as predicate from "../workflow/github_com_widmogrod_mkunion_x_storage_predicate";

export type Cursor = string

export type PaginatedTableState<T> = {
    limit: number
    sort: PaginatedTableSort
    selected: { [key: string]: schemaless.Record<T> }

    prevPage?: Cursor
    nextPage?: Cursor

    where?: predicate.WherePredicates
}

export type PaginatedTableSort = {
    [key: string]: boolean
}

export type PaginatedTableAction<T> = {
    name: string
    action: (state: PaginatedTableState<T>, ctx: PaginatedTableContext<T>) => void
}

export type PaginatedCompare = {
    location: string
    operation: "==" | "!=" | "<" | "<=" | ">" | ">="
    literal: schema.Schema
}

export type PaginatedTableContext<T> = {
    refresh: () => void
    clearSelection: () => void
    filter: (x: predicate.WherePredicates) => void
}


export type PaginatedTableProps<T> = {
    limit?: number
    sort?: PaginatedTableSort
    load: (input: PaginatedTableState<T>) => Promise<schemaless.PageResult<schemaless.Record<T>>>
    mapData?: (data: schemaless.Record<T>, ctx: PaginatedTableContext<T>) => JSX.Element
    actions?: PaginatedTableAction<T>[]
}

export function PaginatedTable<T>(props: PaginatedTableProps<T>) {
    const [data, setData] = useState({Items: [] as any[]} as schemaless.PageResult<schemaless.Record<T>>)
    const [state, setState] = useState({
        limit: props.limit || 3,
        sort: props.sort || {},
        selected: {},
    } as PaginatedTableState<T>)

    useEffect(() => {
        props.load(state).then(setData)
    }, [state, props])

    const ctx = {
        refresh: () => {
            props.load(state).then(setData)
        },
        clearSelection: () => {
            setState({
                ...state,
                selected: {},
            })
        },
        filter: (x: predicate.WherePredicates) => {
            setState({
                ...state,
                where: mergeFilters(state.where, x),
                // always reset the cursor when filtering
                // user can be on a page that doesn't exist anymore
                nextPage: undefined,
                prevPage: undefined
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

    const nextPage = (e: React.MouseEvent) => {
        e.preventDefault()
        setState({
            ...state,
            nextPage: data.Next?.After,
            prevPage: undefined,
        })
    }

    const prevPage = (e: React.MouseEvent) => {
        e.preventDefault()
        setState({
            ...state,
            nextPage: undefined,
            prevPage: data.Prev?.Before,
        })
    }

    return <table>
        <thead>
        <tr>
            <th colSpan={3} className={"option-row"}>
                <button className={"refresh"} onClick={() => ctx.refresh()}>Refresh</button>

                {props.actions && props.actions.map((action) => {
                    return <button
                        key={action.name}
                        className={"action "}
                        disabled={Object.keys(state.selected).length === 0}
                        onClick={applyAction(action)}>{action.name}</button>
                })}

                <WherePredicateRender where={state.where} onChange={(where) => {
                    setState({
                        ...state,
                        where: where,
                    })
                }}/>
            </th>
        </tr>
        <tr>
            <th onClick={batchSelection} className={batchSelectionState()}>
                <button>✧</button>
            </th>
            <th>Prop</th>
            <th>Data</th>
        </tr>
        </thead>
        <tbody>
        {data.Items && data.Items.length > 0 ? data.Items.map((item) => {
            return (
                <tr key={item.ID}>
                    <td><input type={"checkbox"} onChange={selectRowToggle(item)} checked={isSelected(item)}/></td>
                    <td>
                        <dl>
                            <dt onClick={changeSort("ID")} className={sortState("ID")}>ID</dt>
                            <dd>{item.ID}</dd>

                            <dt onClick={changeSort("Type")} className={sortState("Type")}>Type</dt>
                            <dd>{item.Type}</dd>

                            <dt onClick={changeSort("Version")} className={sortState("Version")}>Version</dt>
                            <dd>{item.Version}</dd>
                        </dl>
                    </td>
                    <td>{props.mapData && props.mapData(item, ctx)}</td>
                </tr>
            );
        }) : (
            <tr>
                <td colSpan={5}>No data</td>
            </tr>
        )}
        </tbody>
        <tfoot>
        <tr>
            <td colSpan={3} className={"option-row"}>
                {data.Next && <button onClick={nextPage} className={"next-page"}>Next page</button>}
                {data.Prev && <button onClick={prevPage} className={"prev-page"}>Prev page</button>}
            </td>
        </tr>
        </tfoot>
    </table>
}

function mergeFilters(a?: predicate.WherePredicates, b?: predicate.WherePredicates): predicate.WherePredicates | undefined {
    if (a === undefined) {
        return b
    }
    if (b === undefined) {
        return a
    }

    let result = {...a}
    result.Params = {...a.Params, ...b.Params}
    result.Predicate = mergePredicate(a.Predicate, b.Predicate)

    return result
}

function mergePredicate(a?: predicate.Predicate, b?: predicate.Predicate): predicate.Predicate | undefined {
    if (a === undefined) {
        return b
    }
    if (b === undefined) {
        return a
    }

    return {
        "$type": "predicate.And",
        "predicate.And": {
            L: [a, b],
        },
    }
}


function WherePredicateRender(props: {
    where?: predicate.WherePredicates,
    onChange: (where?: predicate.WherePredicates) => void
}) {
    if (!props.where) {
        return <></>
    }

    return <div className={"filter-rules"}>
        <PredicateRender predicate={props.where.Predicate}
                         onChange={(where) => {
                             if (!where) {
                                 props.onChange(undefined)
                                 return
                             } else {
                                 props.onChange({
                                     ...props.where,
                                     Predicate: where,
                                 })
                             }
                         }}/>
    </div>
}

function PredicateRender(props: {
    predicate?: predicate.Predicate,
    onChange?: (where?: predicate.Predicate) => void
}) {
    if (!props.predicate) {
        return <></>
    }

    switch (props.predicate.$type) {
        case "predicate.And":
            let and = props.predicate["predicate.And"]

            if (!and.L) {
                return <></>
            }

            return <div className={"filter-group"}>
                <button onClick={() => {
                    props.onChange && props.onChange(undefined)
                }}>x
                </button>
                (AND
                {and.L?.map((x) => {
                    return <PredicateRender
                        predicate={x}
                        onChange={(where) => {
                            let predicates = and.L?.map((y) => (y === x) ? where : y).filter((y) => y !== undefined) as predicate.Predicate[]

                            if (predicates.length === 0) {
                                props.onChange && props.onChange(undefined)
                                return
                            }

                            props.onChange && props.onChange({
                                "$type": "predicate.And",
                                "predicate.And": {
                                    L: predicates
                                }
                            })
                        }}
                    />
                })}
                )
            </div>

        case "predicate.Or":
            let or = props.predicate["predicate.Or"]

            if (!or.L) {
                return <></>
            }


            return <div className={"filter-group"}>
                <button onClick={() => {
                    props.onChange && props.onChange(undefined)
                }}>x
                </button>
                (OR
                {or.L?.map((x) => {
                    return <PredicateRender
                        predicate={x}
                        onChange={(where) => {
                            let predicates = or.L?.map((y) => (y === x) ? where : y).filter((y) => y !== undefined) as predicate.Predicate[]

                            if (predicates.length === 0) {
                                props.onChange && props.onChange(undefined)
                                return
                            }

                            props.onChange && props.onChange({
                                "$type": "predicate.Or",
                                "predicate.Or": {
                                    L: predicates
                                }
                            })
                        }}
                    />
                })}
                )
            </div>

        case "predicate.Not":
            let not = props.predicate["predicate.Not"]

            if (!not.P) {
                return <></>
            }

            return <div className={"filter-group"}>
                not <PredicateRender predicate={not.P}/>
            </div>

        case "predicate.Compare":
            let compare = props.predicate["predicate.Compare"]

            return <div className={"filter-rule"}>
                <button onClick={() => {
                    props.onChange && props.onChange(undefined)
                }}>x
                </button>
                <input type={"text"}
                       disabled={true}
                       value={compare.Location}
                />
                <select
                    value={compare.Operation}
                    onChange={(e) => {
                        props.onChange && props.onChange({
                            "$type": "predicate.Compare",
                            "predicate.Compare": {
                                ...compare,
                                Operation: e.target.value
                            }
                        })
                    }}
                >
                    <option value={"=="}>==</option>
                    <option value={"!="}>!=</option>
                    <option value={"<"}>{"<"}</option>
                    <option value={"<="}>{"<="}</option>
                    <option value={">"}>{">"}</option>
                    <option value={">="}>{">="}</option>
                </select>
                <BindableValueRender bindable={compare.BindValue}/>
            </div>
    }

    return <div>Unknown predicate</div>
}

function BindableValueRender(props: { bindable?: predicate.Bindable }) {
    if (!props.bindable) {
        return <></>
    }

    switch (props.bindable.$type) {
        case "predicate.BindValue":
            let value = props.bindable["predicate.BindValue"]

            return <input type={"text"} value={value.BindName}/>

        case "predicate.Literal":
            let literal = props.bindable["predicate.Literal"]

            return <SchemaValue data={literal.Value}/>

        case "predicate.Locatable":
            let locatable = props.bindable["predicate.Locatable"]

            return <div>
                {locatable.Location}
            </div>
    }

    return <div>Unknown bindable {JSON.stringify(props.bindable)}</div>
}

function SchemaValue(props: { data?: schema.Schema }) {
    if (!props.data) {
        return <></>
    }

    switch (props.data.$type) {
        case "schema.String":
            return <input type={"text"} value={props.data["schema.String"]} disabled={true}/>
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
                return <></>; // If the map is empty, return null (no table to display)
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

    return <div>
        Unknown schema {JSON.stringify(props.data)}
    </div>
}