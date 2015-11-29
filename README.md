# go-confidant

This Go package contains functionality to talk to Confidant.


# How to use?

```go
    package main

    import(
      "fmt"
      "github.com/mickelsonm/go-confidant"  
    )

    func main(){
      confidant.Configure(&confidant.GlobalConfig{
        AWSRegion: "us-west-1"  
      })

      result := confidant.GetService(&confidant.Config{
        URL: "https://confidant-production.example.com",
        AuthKey: "TOO_MANY_SECRETS",
        FromContext: "myservice-production",
        ToContext: "confidant-production",
        TokenLife: 1,
      })

      if result.Error != nil{
        fmt.Println(result.Error)
        return
      }

      fmt.Println(string(result.Service))
    }
```

Global configuration properties:

- AWSRegion: region used by KMS

Confidant configuration properties:

- TokenLife: token lifetime in minutes (defaults to 1)
- AuthKey: KMS auth key
- FromContext: IAM role requesting secrets (our client/what uses this)
- ToContext: IAM role of the Confidant server
- URL: the URL of the Confidant server

# How to Contribute?

Please fork this repository, make your code changes, and submit a pull request.

# License

MIT, but see Lyft's licensing for usage of Confidant.
