# x/workflow - manage workflows in transactional way

When you find yourself that you have to do two or more write operation in one operation, you may find this library helpful.

```go
flow RegisterUser(input) {
	reserved = ReserveNick(input.nick) @retry(3)
	if reserved.status != "ok" {
		return({status: "error", reason: "reservation-failed"})
    }
	
	//account = CreateAccount({
	//	nick: reserved.nick,
	//	confirmed: false,
	//	..input
    //}) @retry(3)
	//
    //if account.status != "ok" {
	//    RemoveReservation(reserved.ID) @retry(3)
	//	return({status: "error", reason: "account-creation-problem"})
    //}
	
	let confirmed = await SendActivationEmail(account.email) @timeout(24h) @retry(3)
	if confirmed.status != "ok" {
		//DisableAccount()
		RemoveReservation(reserved.ID) @retry(3)
		return({status: "error", reason: "email-not-confirmed-withing"})
    }

    account = CreateAccount({
        nick: reserved.nick,
        ..input
    }) @retry(3)
	
	return ({status: "ok", id: account.id})
}
```

```
sdk {
    flow SendActivationEmail(input) {
       user, err = FindUser(select{activationState}, where{activation.code=code, activation.active=none})
       newState, err = Machine(state).Handle(&StartActivation{})
       err = UpdateUser(user.id, {activation=newState}) 
       
       let res = mandril.Send(temaplate, newState.ActivationCode)
       if res.status != "ok" {
          Machine(state).Handle
       }
    }
    machine ActivateWithCode(code) {
        user, err = FindUser(select{user, activationState}, where{activation.code=code, activation.active=none})
        newState, err = Machine(state).Handle(&ConfirmActivatio{code})
        err = UpdateUser(user.id, {activation=newState})
    }
}
```

```
machine channels {
    NewCommChannel{
        userId="",
        varified
        type = "email" | "phoneNo" | "push"
        email {address}
        phone {number}
        push {ios, appnid}
    }
    Confirm{
        code="",
    }
}

sdk.NewCommChannel{
    input={userId=123, type="email", email="asdf@asfd.com"},
    OnOk(channel)={
        res = SendEmail(input.email, template, input.code),
    }
}

sdk.Confirm{
    code={code},
    OnOk(channel)={
        UpdateUser({id=input.userId, })
    }
}
```

```
Moderation
- automatic, for content that was created by organic users
- many automatic algorithms, text, toxicity, spam, 
- for images, 
- some could depend on language, 


sdk.CreateQuestion(
    input={},
    OnOn(question) {
        AutomaticModeration() {
            let text = aws.TextModeration()
            let sen = google.Sentiment()
            let asd = custom.Toxcity()
            
            UpdateQuestion({
                "moderationText": {...text},
                "sen": {...sen},
                "asd", {...asd},
            })
        }
    }
)

sdk.CreateQuestion(
    input={
        disableAutomaticModeration=...
    },
)

sdk.TaskQueue({
    type: ["created"],
    where: "question",
    run: "AutomaticModeration",
})

```

```go
for _, flow := range flows {
	switch flow.state {
	case "Error":
		// return
    case "Callback":
		// ...
    }
}
```

- Functions must be idempotent. Runtime will always provide unique to operation call ID, so function or middleware needs to deduplicate it

## Roadmap
### Ideas
- [ ] proof of concept with simple ui to explore failed states 
- [ ] generate from program logic, functions to be implemented
- [ ] deployment planner, allow to deploy function as embedded (wasm) or remote with autoscaling rules
- [ ] background processing logic (pull base)
- [ ] background processing logic stream based
- [ ] generate functions from OpenAPI, gRPC spec
- [ ] lang server with autocomplete