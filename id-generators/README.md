# ID Generators 

## Disk + memory based for single instance 

## Amazon way of doing id generation

## UUID Based generation in MySQL 

### Time comparison 

UUID based table writes - 229.242056458 seconds
INT based table writes - 214.075277625 seconds

### Size comparison 
-- ---------------+------------+------------+-----------+
| database_name | table_name | index_name | Size (KB) |
+---------------+------------+------------+-----------+
| id_generator  | uuid_demo  | PRIMARY    |  86800.00 |
| id_generator  | uuid_demo  | idx_age    |  48784.00 |
-- ---------------+------------+------------+-----------+

+---------------+------------+------------+-----------+
| database_name | table_name | index_name | Size (KB) |
+---------------+------------+------------+-----------+
| id_generator  | int_demo   | PRIMARY    |  26160.00 |
| id_generator  | int_demo   | idx_age    |  15888.00 |
+---------------+------------+------------+-----------+

## MongoDB based objectID 

## Snowflake via API and Stored Procedure

## Flickr odd-even based ID Generation 

