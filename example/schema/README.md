# Golang recursive schema
Library allows to write code that work with any type of schemas.
Regardless if those are JSON, XML, YAML, or golang structs.

Most benefits
- Union types can be deserialized into interface field

## TODO
- [ ] Support json tags in golang to map field names to schema
- [ ] Support JSON float64 since value is `any` which is not always true. 
      Value should be split into Int, Float, String, Bool, and Null
- [ ] Add cata, ana, and hylo morphisms

