package dynamo

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type DynamoDB struct {
	dbConn          *sql.DB
	tableAttributes map[string]DynamoDBTableAttributes
}

func NewDynamoDB() *DynamoDB {
	db, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/dynamodb?charset=utf8")
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to DynamoDB (MySQL): %v", err))
	}

	return &DynamoDB{
		dbConn:          db,
		tableAttributes: make(map[string]DynamoDBTableAttributes),
	}
}

func (d *DynamoDB) CreateTable(input CreateTableInput) error {
	tableName := input.TableName
	partitionKey := input.Keys.PartitionKey
	sortKey := input.Keys.SortKey

	lsis := make([]LocalSecondaryIndex, 0)

	statement := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ( %s VARCHAR(255) NOT NULL,", tableName, partitionKey)
	if sortKey != nil {
		statement += fmt.Sprintf("%s VARCHAR(255) NOT NULL,", sortKey)
	}

	// Construct the CREATE TABLE statement
	for _, attr := range input.Attributes {
		if attr != partitionKey && (sortKey == nil || attr != sortKey) {
			statement += fmt.Sprintf("%s VARCHAR(255),", attr)
		}
	}

	if sortKey != nil {
		statement += fmt.Sprintf(" PRIMARY KEY (%s, %s)", partitionKey, sortKey)
	} else {
		statement += fmt.Sprintf(" PRIMARY KEY (%s)", partitionKey)
	}

	if len(input.LSIs) > 0 {
		for _, lsi := range input.LSIs {
			statement += fmt.Sprintf(", INDEX %s (%s, %s)", lsi.IndexName, partitionKey, lsi.SortKey)
		}
		lsis = input.LSIs
	}
	statement += ");"

	// fmt.Println("Create table statement: ", statement)

	result, err := d.dbConn.Exec(statement)
	if err != nil {
		fmt.Printf("Failed to create table %s: %v\n", tableName, err)
		return err
	}
	// fmt.Printf("Created table: %s, details: %v", tableName, result)
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Table %s created or already exists, rows affected: %d\n", tableName, rowsAffected)

	d.tableAttributes[tableName] = DynamoDBTableAttributes{
		TableName:     tableName,
		KeyAttributes: input.Keys,
		LSIs:          lsis,
	}

	return nil
}

func (d *DynamoDB) PutItem(input PutItemInput) error {

	tableName := input.TableName
	if tableName == "" {
		return fmt.Errorf("table name is required")
	}

	value, ok := d.tableAttributes[tableName]
	if !ok {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	partitionKeyAttr := value.KeyAttributes.PartitionKey
	sortKeyAttr := value.KeyAttributes.SortKey

	if input.KeyAttributes[partitionKeyAttr] == "" {
		return fmt.Errorf("partition key %s is required", partitionKeyAttr)
	}

	if sortKeyAttr != nil && input.KeyAttributes[sortKeyAttr.(string)] == "" {
		return fmt.Errorf("sort key %s is required for this table", sortKeyAttr)
	}

	// Build columns, placeholders and args for a parameterized INSERT
	columns := make([]string, 0, 2+len(input.Values))
	placeholders := make([]string, 0, 2+len(input.Values))
	args := make([]interface{}, 0, 2+len(input.Values))

	// add partition key
	columns = append(columns, partitionKeyAttr)
	placeholders = append(placeholders, "?")
	args = append(args, input.KeyAttributes[partitionKeyAttr])

	// add sort key if present
	if sortKeyAttr != nil {
		columns = append(columns, sortKeyAttr.(string))
		placeholders = append(placeholders, "?")
		args = append(args, input.KeyAttributes[sortKeyAttr.(string)])
	}

	// add other values (map iteration order is not guaranteed; that's acceptable here)
	for k, v := range input.Values {
		columns = append(columns, k)
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}

	statement := fmt.Sprintf(
		"REPLACE INTO %s (%s) VALUES (%s);",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// fmt.Println("Put item statement: ", statement, " args:", args)

	result, err := d.dbConn.Exec(statement, args...)
	if err != nil {
		fmt.Printf("Failed to insert item into table %s: %v\n", tableName, err)
		return err
	}
	fmt.Printf("Inserted item into table %s, result: %v\n", tableName, result)
	return nil
}

func (d *DynamoDB) GetItem(input GetItemInput) (*GetItemOutput, error) {
	if input.TableName == "" || input.PartitionKey == "" {
		return nil, fmt.Errorf("table name and partition key are required")
	}

	tableName := input.TableName

	keyAttrs, ok := d.tableAttributes[tableName]
	if !ok {
		return nil, fmt.Errorf("table %s does not exist", tableName)
	}

	partitionKeyAttr := keyAttrs.KeyAttributes.PartitionKey
	sortKeyAttr := keyAttrs.KeyAttributes.SortKey

	if sortKeyAttr != nil && input.SortKey == "" {
		return nil, fmt.Errorf("sort key %s is required for this table", sortKeyAttr)
	}

	// Build the SELECT statement and args
	args := make([]interface{}, 0, 2)
	args = append(args, input.PartitionKey)
	if sortKeyAttr != nil {
		args = append(args, input.SortKey)
	}
	statement := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", input.ProjectionExpression, input.TableName, partitionKeyAttr)
	if sortKeyAttr != nil {
		statement += fmt.Sprintf(" AND %s = ?", sortKeyAttr)
	}

	// fmt.Println("Get item statement: ", statement, " args:", args)

	// Execute the query
	row := d.dbConn.QueryRow(statement, args...)
	if row == nil || row.Err() != nil {
		fmt.Printf("Failed to get item from table %s\n", tableName)
		return nil, errors.New("no rows found")
	}

	// Prepare to scan the result
	columns := strings.Split(input.ProjectionExpression, ",")
	values := make([]sql.NullString, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	err := row.Scan(valuePtrs...)
	if err != nil {
		fmt.Printf("Failed to scan item from table %s: %v\n", tableName, err)
		return nil, err
	}

	resultValues := make(map[string]string)
	for i, col := range columns {
		if values[i].Valid {
			resultValues[strings.TrimSpace(col)] = values[i].String
		} else {
			resultValues[strings.TrimSpace(col)] = ""
		}
	}

	fmt.Printf("Retrieved item from table %s: %v\n", tableName, resultValues)

	return &GetItemOutput{
		Values: resultValues,
	}, nil

}

func (d *DynamoDB) Query(input QueryInput) (*QueryOutput, error) {
	if input.TableName == "" || input.KeyConditionExpression == "" {
		return nil, fmt.Errorf("table name and partition key are required")
	}

	tableName := input.TableName

	attrs, ok := d.tableAttributes[tableName]
	if !ok {
		return nil, fmt.Errorf("table %s does not exist", tableName)
	}

	keyAttrs := attrs.KeyAttributes

	// 1. Determine the effective sort key for ordering.
	var effectiveSortKey string

	if input.IndexName != nil && input.IndexName != "" {
		// Assume IndexName is an LSI name.
		indexNameStr := input.IndexName.(string)
		found := false
		for _, lsi := range attrs.LSIs {
			if lsi.IndexName == indexNameStr {
				effectiveSortKey = lsi.SortKey // Use LSI's SortKey
				found = true
				break
			}
		}
		if !found {
			// A true DynamoDB implementation would check GSI/LSI existence.
			return nil, fmt.Errorf("index %s does not exist on table %s (or is not a registered LSI)", indexNameStr, tableName)
		}
	} else if keyAttrs.SortKey != nil {
		// Use the primary SortKey if no index is specified.
		effectiveSortKey = keyAttrs.SortKey.(string)
	}

	// Build the SELECT statement and args
	args := make([]interface{}, 0, len(input.KeyValues))
	for _, v := range input.KeyValues {
		args = append(args, v)
	}
	statement := fmt.Sprintf("SELECT %s FROM %s WHERE %s", input.ProjectionExpression, input.TableName, input.KeyConditionExpression)
	if effectiveSortKey != "" {
		if !input.ScanIndexForward {
			statement += " ORDER BY " + effectiveSortKey + " DESC"
		} else {
			statement += " ORDER BY " + effectiveSortKey + " ASC"
		}
	}

	// fmt.Println("Query statement: ", statement, " args:", args)

	// Execute the query
	rows, err := d.dbConn.Query(statement, args...)
	if err != nil {
		fmt.Printf("Failed to query items from table %s: %v\n", tableName, err)
		return nil, err
	}
	defer rows.Close()

	// Prepare to scan the results
	columns := strings.Split(input.ProjectionExpression, ",")
	resultItems := make([]map[string]string, 0)

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			fmt.Printf("Failed to scan item from table %s: %v\n", tableName, err)
			return nil, err
		}

		item := make(map[string]string)
		for i, col := range columns {
			if values[i].Valid {
				item[strings.TrimSpace(col)] = values[i].String
			} else {
				item[strings.TrimSpace(col)] = ""
			}
		}
		resultItems = append(resultItems, item)
	}

	fmt.Printf("Queried items from table %s: %v\n", tableName, resultItems)

	return &QueryOutput{
		Items: resultItems,
	}, nil
}
