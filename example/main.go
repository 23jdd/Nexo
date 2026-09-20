package main

import (
	"fmt"

	"github.com/23jdd/Nexo/protocol/http1"
	"github.com/23jdd/Nexo/server"
)

func main() {
	httpServer := server.New()
	httpServer.HandleFunc("/", func(w *http1.ResponseWriter, r *http1.Request) {
		name := r.Query("name")
		fmt.Println("name:" + name)
		w.Write([]byte(`<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nexo</title>
    <style>
          div {
              color:black;
              text-align: center;
              font-size: 30px;
          }
    </style>
</head>
<body>
   <div>Hello Nexo</div>
</body>
</html>`))
	})
	if err := httpServer.Run("127.0.0.1", 9999); err != nil {
		panic(err)
	}
}
