// =======================================================================================================
// Title: awsenvsec
// Description: Retrieve AWS Secrets as Environment Variables from Secrets Manager and/or Parameter Store
// =======================================================================================================

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
)

// Header variables and description
var t = time.Now()
var title = "awsenvsec"
var description = "Retrieve AWS Secrets as Environment Variables from Secrets Manager and/or Parameter Store"
var date = t.Format("2006-01-02 15:04:05")

// Header function
func header(title, curDate, description string) string {
	hlines := strings.Repeat("-", 102)
	header := hlines + "\n" +
		"Script: " + title + "\n" +
		"Description: " + description + "\n" +
		"Date: " + curDate + "\n" +
		hlines + "\n"
	return header
}

// Flags and defaults
var (
	profileFlag   = flag.String("profile", "", "AWS Profile (optional)")
	regionFlag    = flag.String("region", "", "AWS Region")
	outputFlag    = flag.String("output", "", "Output: JSON or ENV")
	smPathFlag    = flag.String("smpath", "", "AWS Secrets Manager Path")
	psPathFlag    = flag.String("pspath", "", "AWS Parameter Store Path")
	recursiveFlag = flag.Bool("recursive", false, "Recursive Flag")
	defaultRegion = "us-east-1"
	defaultSmPath = ""
	defaultPsPath = ""
)

// Initialize an empty usage() function used in init function
func usage() {}

// Extend flags for short forms with custom usage
func init() {

	// Look up existing ENV variables, use them as defaults before parsing flags
	envRegion, ok := os.LookupEnv("AWS_REGION")
	if ok {
		defaultRegion = envRegion
	}

	envSmPath, ok := os.LookupEnv("SM_PATH")
	if ok {
		defaultSmPath = envSmPath
	}

	envPsPath, ok := os.LookupEnv("PS_PATH")
	if ok {
		defaultPsPath = envPsPath
	}

	flag.StringVar(profileFlag, "p", "", "AWS Profile (optional)")
	flag.StringVar(regionFlag, "r", defaultRegion, "AWS Region")
	flag.StringVar(outputFlag, "o", "", "Output to environment file (optional: must be either \"env\" or \"json\"")
	flag.StringVar(smPathFlag, "sm", defaultSmPath, "AWS Secrets Manager Path")
	flag.StringVar(psPathFlag, "ps", defaultPsPath, "AWS Secrets Manager Path")
	flag.BoolVar(recursiveFlag, "c", false, "Recursive Flag")

	flag.Usage = func() {
		fmt.Println(("Usage: ") + os.Args[0] + " [-p profile] [-r region] [-sm smpath] [-ps pspath] [-c recursive] [-o output]")

		fmt.Println("\nFlags:")
		flagOptions := "  -p  | --profile         AWS Profile (Profile Name - Optional)\n" +
			"  -r  | --region          AWS Region (Default: us-east-1)\n" +
			"  -sm | --smpath          AWS Secrets Manager Path (example: \"product/dev/var\")\n" +
			"  -ps | --pspath          AWS Parameter Store Path (example: \"/product/dev/var\")\n" +
			"  -c  | --recursive       Recursive Flag (used if recursion is needed in Parameter Store)\n" +
			"  -o  | --output          Output to environment file (optional: must be either \"env\" or \"json\")\n"

		fmt.Println(flagOptions)
		fmt.Println("Example: " + os.Args[0] + " -p myprofile -r us-east-1 -sm \"product/dev/var\" -ps \"/product/dev/var\" -c -o json\n")
	}
}

// Check if incoming JSON string is valid
func isItJSON(jsonString string) bool {
	var rawJson json.RawMessage
	return json.Unmarshal([]byte(jsonString), &rawJson) == nil
}

func main() {

	// Parse flags: suppress error if missing argument after flag.
	// This isn't required but provides a "cleaner" output.
	flag.CommandLine.SetOutput(ioutil.Discard)
	flag.Parse()

	// If output flag exists, it can only have two values
	if *outputFlag != "" {

		// Print header when running in cli mode (with output)
		fmt.Println(header(title, date, description))

		fmt.Println("Output flag used: " + *outputFlag + ", running as a cli app...")
		if (*outputFlag != "env") && (*outputFlag != "json") {
			fmt.Println("There was a problem running " + os.Args[0] + "\n")
			flag.Usage()
			fmt.Println("Output flag was: " + *outputFlag + ". It must be either \"env\" or \"json\"")
			os.Exit(2)
		}
	}

	// Create a map to store results
	resultsMap := make(map[string]string)

	// Secrets Manager: Decrypt secrets and loop through key/value JSON if it exists
	if len(*smPathFlag) > 0 {
		secrets := getAllSecrets()
		if secrets != nil {
			for _, s := range secrets {
				var secretsMap map[string]interface{}
				secretJson := decryptSecret(s.Name)
				// Check if result is JSON or Plaintext
				if isItJSON(secretJson) {
					err := json.Unmarshal([]byte(secretJson), &secretsMap)
					if err != nil {
						fmt.Println(err)
					}
					for k, v := range secretsMap {
						k = strings.ToUpper(k)
						val := fmt.Sprintf("%v", v)
						resultsMap[k] = val
					}
				} else {
					k := strings.ToUpper(fmt.Sprintf(s.Name[strings.LastIndex(s.Name, "/")+1:]))
					v := decryptSecret(s.Name)
					resultsMap[k] = v
				}
			}
		} else {
			fmt.Println("There was an issue retrieving secrets from AWS Secrets Manager.")
		}
	}

	// Parameter Store: Retrieve decrypted results, used with -c flag for recursive path lookups
	if len(*psPathFlag) > 0 {
		parameters := getAllParameters()

		if parameters != nil {
			for _, p := range parameters {
				param := strings.ToUpper(fmt.Sprintf(p.Name[strings.LastIndex(p.Name, "/")+1:]))
				resultsMap[param] = p.Value
			}
		} else {
			fmt.Println("There was an issue retrieving secrets from AWS Parameter Store. Please check the path.")
		}
	}

	// Output cases when running in cli mode
	switch *outputFlag {
	case "env":
		resultsMapSorted := make([]string, 0, len(resultsMap))
		for k := range resultsMap {
			resultsMapSorted = append(resultsMapSorted, k)
		}
		sort.Strings(resultsMapSorted)
		fmt.Println("\n# Environment Variables: " + date + "\n")
		for _, k := range resultsMapSorted {
			fmt.Println(k + "=" + "\"" + resultsMap[k] + "\"")
		}
		os.Exit(0)
	case "json":
		// Serialize the map into JSON
		jsonOutput, err := json.Marshal(resultsMap)
		if err != nil {
			fmt.Println(err)
		}
		// "Pretty" print the JSON for a nicer, indented output
		var prettyJSON bytes.Buffer
		error := json.Indent(&prettyJSON, []byte(jsonOutput), "", "   ")
		if error != nil {
			fmt.Println(" ")
			return
		}
		fmt.Println("\n# Environment Variables: " + date + "\n")
		fmt.Print(string(prettyJSON.Bytes()), "\n")
		os.Exit(0)
	}

	// Export the appended map results for use as AWS environment secret variables
	for k, v := range resultsMap {
		fmt.Printf("export %s=$\"%s\"\n", k, v)
	}

}
