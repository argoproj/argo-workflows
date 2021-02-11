

cat ./docs/environment-variables.md \
  | grep "| \`" \
  | awk '{gsub(/\`/, "", $2);  print $2; }' \
  | while read -r x; do
    if ! grep -qR $x ./workflow ./persist ./util; then
      echo "Documented variable $x is not used anywhere";
      exit 1;
    fi;
  done

