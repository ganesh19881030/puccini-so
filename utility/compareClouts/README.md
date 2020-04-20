You can build the compareClouts utility by issuing the following command in the compareClouts folder:

	go build

This should create a compareClouts.exe which you can use to compare either clout.yaml or clout.json files

In order to run the compare utility, use the following command:

	./compareClouts.exe clout1.yaml clout2.yaml
	
or

	./compareClouts.exe clout1.json clout2.json

  e.g. 

    	./compareClouts.exe C:/firewall_service_1.json C:/firewall_service_2.json
	
or if you want to run the utility without building it, you can use the following command:

	go run compare.go logging.go clout1.yaml clout2.yaml

	
It generates a log file called compare-clout.log which logs the comparison based on
the log level configured. By default the log level is INFO (1)
	
You can specify the log level with -log n option as shown below:

	./compareClouts.exe clout1.yaml clout2.yaml -log n
  
  where n is:
  	
	  	= 0 for ERROR
  
          	= 1 for INFO
		
		= 2 for DEBUG
