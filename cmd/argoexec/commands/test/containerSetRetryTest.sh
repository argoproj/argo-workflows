FILE=test.txt
if test -f "$FILE"; then
    echo "$FILE exists."
    exit 0
else 
    touch $FILE
    echo "$FILE don't exists."
    exit 1
fi