package http_utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func Marshal(i interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(i)
	return bytes.TrimRight(buffer.Bytes(), "\n"), err
}

type ReqHeader struct {
	HeaderName  string
	HeaderValue string
}

/*
A wrapper for Http Requests

  - method <string> : request method i.e. GET, POST etc

  - payload <interface{}> : Struct for json marshaling into data payload

  - reqHeaders <[]ReqHeader> : defaults to "Content-Type: application/json; charset=utf-8" and "Accept: application/json" if nil

  - addHeaders <[]ReqHeader> : nil or slice of additional <ReqHeader> if you want to add to the default headers

Returns :
  - response body as []byte
  - response.Status as response code string or empty string on error
  - error
*/
func HttpPostReq(method string, payload interface{}, url string, reqHeaders []ReqHeader, addHeaders []ReqHeader) ([]byte, string, error) {
	if reqHeaders == nil {
		defaultHeader := []ReqHeader{
			{HeaderName: "Content-Type", HeaderValue: "application/json; charset=utf-8"},
			{HeaderName: "Accept", HeaderValue: "application/json"},
		}
		reqHeaders = defaultHeader
	}
	if addHeaders != nil {
		reqHeaders = append(reqHeaders, addHeaders...)
	}
	var returnByes []byte
	var reqBytes []byte
	var err error
	if payload != nil {
		reqBytes, err = json.Marshal(&payload)
		if err != nil {
			return returnByes, "", err
		}
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return returnByes, "", err
	}

	for i := 0; i < len(reqHeaders); i++ {
		request.Header.Set(reqHeaders[i].HeaderName, reqHeaders[i].HeaderValue)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return returnByes, "", err
	}

	defer response.Body.Close()
	rBody, err := io.ReadAll(response.Body)
	if err != nil {
		return returnByes, "", err
	}

	return rBody, response.Status, nil
}

/*
Decodes json from an incoming request body to an object interface{}
*/
func GetReqFromJSON(r *http.Request, reqObj interface{}) error {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqObj)
	if err != nil {
		return err
	}
	return nil
}

/* Camel case to snake case */
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

/* Converts CamelCase to snake-case */
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
}

func GetFieldType(field reflect.Value) string {
	return fmt.Sprintf("%s", field.Type())
}

func GetAndAppendQueries(rawValue interface{}, fieldTypeString string, fieldNameString string, queries *[]string) {
	switch fieldTypeString {
	case "*[]string":

		var sli *[]string = rawValue.(*[]string) //type assert raw Field.Interface() to *[]string
		if sli != nil && len(*sli) > 0 {
			var subSli []string
			for _, str := range *sli {
				subSli = append(subSli, fieldNameString+"[]="+str)
			}
			*queries = append(*queries, subSli...)
		}

	case "*string":
		var qStr string
		var str *string = rawValue.(*string) //type assert raw Field.Interface() to *string
		qStr += fieldNameString + "=" + *str
		*queries = append(*queries, qStr)

	case "*int":
		var qStr string
		var numb *int = rawValue.(*int)
		numbStr := strconv.Itoa(*numb)
		qStr += fieldNameString + "=" + numbStr

	case "*int32":
		var qStr string
		var numb *int32 = rawValue.(*int32)
		numbStr := strconv.FormatInt(int64(*numb), 10)
		qStr += fieldNameString + "=" + numbStr

	case "*int64":
		var qStr string
		var numb *int64 = rawValue.(*int64)
		numbStr := strconv.FormatInt(*numb, 10)
		qStr += fieldNameString + "=" + numbStr

	case "*big.Int":
		var qStr string
		var numb *big.Int = rawValue.(*big.Int)
		numbStr := numb.String()
		qStr += fieldNameString + "=" + numbStr

	case "*bool":
		//do string array
		var qStr string
		var str *bool = rawValue.(*bool) //type assert raw Field.Interface() to *string
		qStr += fieldNameString + "=" + strconv.FormatBool(*str)
		*queries = append(*queries, qStr)
	}
}

/*
-	Req struct should only have *string, *[]string, *int, *int32, *int64, *big.Int, and *bool
-	Pointers only so we can check for absence with nil
-	Since GET query params are always strings, the safest best is to only work with request structs onf type *string
-	Req fields should all be CamelCase, to be translated into snake-case for the queryparam keys
-	req <interface{}> : The provided get request struct i.e. {"QueryParamOne": "true", "QueryParamTwo":"TSLA"}
*/
func RequestStructToquery(req interface{}) string {
	var queries []string
	val := reflect.ValueOf(req)
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {

		field := val.Field(i)
		if !field.IsNil() {
			// if the field is not nil, we will process it

			fieldTypeString := GetFieldType(field) // "*[]string", "*bool", etc, so we know how to process the value

			fieldType := typ.Field(i)
			fieldNameStringCamel := fieldType.Name // "SomeQueryParam", so we know how to make the ?query-param key

			fieldNameStringSnake := ToSnakeCase(fieldNameStringCamel)

			GetAndAppendQueries(field.Interface(), fieldTypeString, fieldNameStringSnake, &queries)
		}
	}
	qStr1 := strings.Join(queries, "&")
	qStr0 := "?" + qStr1

	return qStr0

}

func createThing[T any]() T {
	var value T
	return value
}

func DeepEqual[T any](s T) bool {
	t := createThing[T]()
	isEmpty := reflect.DeepEqual(s, t)
	return isEmpty
}
