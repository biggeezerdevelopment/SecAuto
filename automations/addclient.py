import json
import sys

def main():    
    client = context.get('client', 'default-client')
    result = {"clientname": client}
    # Return result as JSON
    return_context(result)

if __name__ == "__main__":
    main()