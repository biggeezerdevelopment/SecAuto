import json

def main():
    #count = context.get('ServiceResponse').get('count').get('#text')
    count = context.get('count')
    ncount = {"qualys_asset_count":int(count)}
    set_cache(f"qualys_asset_count_{context.get('clientname')}", json.dumps(ncount))
    return_context(ncount)

if __name__ == "__main__":
    main()
