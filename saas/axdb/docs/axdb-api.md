# The AXDB REST API spec

The API is specified relative to the root URL, which is determined at deployment time. During development and testing we use http, later we will switch to https.

We will use http GET calls for information retrieval, and http POST calls for updating. 

The server will always return a non-empty JSON body to the client. The returned object will always contain a status value, which is specified below.

## Constants

### HTTP status code
Only 7 status code would be used in API.

* `200:Response to a successful GET, PUT, POST or DELETE.`
* `400:The request is malformed, such as if the body is missing or not unable to parse.`
* `401:When no or invalid authentication details are provided.`
* `403:When creating an object that exists already.`
* `404:When a non-existent resource is requested.`
* `500:The server encountered an unexpected condition which prevented it from fulfilling the request.`

## Objects

### axerror
* code: The unique ax error code, it would a succinct human meaning string.
* message: The human readable summary of the error.
* detail: Detail machine friendly structured message.(JSON)

        * example:
        
```
{
    "code": "ERR_AXDB_INSERT_DUPLICATE",
    "message": "The request tries to insert an entry that already exists",
    "detail": ""
}
```

## Endpoint

Normally axdb is run in a containter. AXDB binary in the container uses port 8080, and you need to map the port to a host port. In this example we use port 9000.

docker run --rm -p 8080:9000 <repo_name>/axdb /axdb/bin/axdb_server

To verify, access http://<host_ip>:9000/v1/axdb/version

## API

### GET axdb/version
* Get the axdb version
* Returns a JSON object

### POST axdb/create_table
* Create a table
* Input: JSON object that represents a table.
* Output:
  * null if successful
  * axerror object if not successful

### GET app_name/table_name?column_name=column_value&...
* Query the DB.
* Input:
	* app_name
	* table_name
	* column_name: the column you are searching, equivalent to where clause in SQL
	* column_value: the column value you are searching

### POST app_name/table_name
* Create a new object in the specified table.
* Input in JSON body:
	* The object that you are creating in the DB. The object's fields should match the table's columns.
* Output:
	* empty json object if successful
	* axerror object if failure

### PUT app_name/table_name
* Update an object in the specified table. We support two different update modes:
    * Normal Update: We will only update the fields that you have specified, and leave the other fields unchanged in DB. If the object to be updated doesn't exist in the DB, a new object will be created using the specified values.
    * Conditional Update: The specified fields will be updated only when the condition associated with Update statement is true.
        * To test whether the row exists in DB before the update operation is performed, add a keyword item "ax_update_if_exist" in JSON body with the value empty.
            * The row is update only when it has existed in DB and the syntax of update statement is valid.
            * An axerror object is returned with relevant message if update statement isn't well-formed. In this case no change to the object will be made.
            * An axerror object is returned with relevant message if update statement is well-formed, but the row to be updated doesn't exist in DB. In this case no new row will be inserted.
        * To test some predicates before the update operation is performed, add a set of key-value pairs (colName : val) to the JSON body. For equality test on a column, append a suffix "_update_if" to the name of the column; for inequality test on a column, append a suffix "_update_ifnot" to the name of the column.
            * The row is update only when all key-value pairs are evaluated to true.
            * An axerror object is returned with relevant message if update statement isn't well-formed. In this case no change to the object will be made.
            * An axerror object is returned with relevant message if update statement is well-formed, but the predicates are evaluated to false. In this case no change will be made.

        * "ax_update_if_exist" and "xxx_update_if" cannot co-exist in JSON body; otherwise an axerror object will be returned with relevant message.
        * The column used with "_update_if" cannot be a part of primary key; otherwise an axerror object will be returned with relevant message.


        ```
        	example 1: Only when the row to be update exists, it will be updated; otherwise, no change will be made.
        	UPDATE table t1 set col1 = xxx1, col2 = xxx2 WHERE pk = ... IF EXISTS

        	example 2: Only when the IF condition is true, the row will be updated.
        	UPDATE table t1 set col1 = xxx1, col2 = xxx2 WHERE pk= ... IF col1_update_if = yyy1 [ AND ...]

        	example 3: The IF condition contains inequality predicate
        	UPDATE table t1 set col1 = xxx1 where pk = ... IF col1_update_ifnot != yyy1 [ AND ...]
        ```


* Input in JSON body:
    * The object you are updating in the DB.
    * The user of AXDB API must follow "ax_update_if_exist" or "_update_if/_update_ifnot" protocol to use conditional update.


* Output:
    * empty json object if successful, which is applicable for both normal update and conditional update.
    * axerror object if any failure, or the put operation isn't performed for conditional update.

### DELETE app_name/table_name
* Delete an object in the specified table.
* Input in JSON body:
	* An array of objects that you are deleting in the DB. You only needs to specify the primary key of the object.
	* If an empty body is given, the whole table will be deleted.
* Output:
	* empty json object if successful
	* axerror object if failure

