#!/bin/bash

# shopt requires bash

echo "Started!"
echo "------"

# First arg to script is the folder of wikifiles
cd $1

apply_to_command (){
	# first argument is the real filename
	# second argument is the decoded filename
	
	echo "> $2"
	echo "From $1"
	# php importTextFile.php "$1" --title "$2" --nooverwrite
	echo "------"
}

# Fancy thing to remove "*.wiki" if can't find anything
# this needs bash
shopt -s nullglob

# Get all files in current directory
FILES=*.wiki

for filename in $FILES
do
	# Replace caret with "\"
	decoded=${filename//^/\\};

	# Replace pile of poo emoji with ":"
	decoded=${filename//💩/:}

	# Replace "_" with "/"
	decoded=${decoded//_/\/};

	# Remove .wiki from title
	decoded=${decoded/.wiki/};

	# Now actually decode it
	decoded=$(printf '%s' "$decoded");

	apply_to_command "$filename" "$decoded";
done

echo "Done!"