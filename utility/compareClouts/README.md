You can use the compare.go utility to compare clout.yaml or clout.json files
In order to run the compare.go, use the commands:

  go run compare.go logging.go clout1.yaml clout2.yaml

or

  go run compare.go logging.go clout1.json clout2.json

  e.g. 

    go run compare.go logging.go C:/firewall_service_1.json C:/firewall_service_2.json
	
It generates a log file called compare-clout.log which logs the comparison based on
the log level configured. By default the log level is INFO (1)
	
You can specify the log level with -log n option as shown below:

  go run compare.go logging.go clout1.yaml clout2.yaml -log n
  
  where n is:
  	
	  	= 0 for ERROR
  
          	= 1 for INFO
		
		= 2 for DEBUG
