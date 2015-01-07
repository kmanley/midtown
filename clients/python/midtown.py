import requests

def main(): 
	r = requests.get("http://localhost:9997/workers")
	print r.status_code
	print r.json()
	print "-" * 40
	r = requests.get("http://localhost:9997/job/150107040846920585")
	print r.status_code
	print r.json()

if __name__ == "__main__":
	main() # TODO: