package shared

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
)

func RandSeq(len int) string {
	// Generates a random sequence of characters of fixed length.
	// Used to generate a username when the user doesn't enter one
	buff := make([]byte, len)
	rand.Read(buff)
	str := base64.StdEncoding.EncodeToString(buff)
	// Base 64 can be longer than len
	return str[:len]
}

func CheckError(err error) {

	// Generic error checking. This function should be called
	// to check and print errors, and exit the program if an error
	// has occured.

	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

}

// Other shared function definitions, if required
