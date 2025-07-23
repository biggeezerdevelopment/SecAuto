import json

def main():
    client = context.get('client') 
    r = {"clientname":client}
    return_context(r)
main()
