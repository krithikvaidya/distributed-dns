# Client-Server Chat Application in Golang

General guidelines:
- Use *defer* wherever appropriate (unlocking a mutex after a function completes execution, writing to a channel at the end of a function's execution, closing a connection, etc.)
- Try to comment and lint your code
- While writing the code, try to account for different types of synchronization issues and failures that may occur.
- You should initialize the project as a go module by using ```go mod init <repository_path>```. For example, if you are storing the project files in a Github repo and your username is *username* and the repository name is *repo*, you will run ```go mod init github.com/username/repo``` in the root folder of the project. Refer to [this](https://golang.org/doc/code.html#Introduction) for more details
- Ideally the program should be able to be compiled the given source code by running ```go build .``` in the source code's root directory and then run using ```./<executable_name> <command_line_params>```.
- A fixed message size for every TCP communication has been assumed. You can also implement it with variable size messages.
- The guidelines written as comments and the functions, etc. are advisory. Feel free to implement it however you see fit.
