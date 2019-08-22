
# SvcWrapper 0.1
Windows Service Wrapper

    - For windows programs to run as services, they have to be developed in a certain way to respond to 
      certain messages from the Windows OS ... otherwise windows kills them after 30 seconds.
    - SvcWrapper wraps "normal" (non-service) Windows executables enabling them to run as a service.
    - It enables "installing", "removing" and "running" applications as services.
    
    Usage: wrapper.exe SERVICE_NAME COMMAND [ARGS]
        - SERVICE_NAME is the name of the service
        - COMMAND is either 'install', 'remove' or 'run'

            Example 01:
            - > wrapper.exe kolide_launcher_service install launcher.exe --host=192.168.0.10 --insecure
                This will create a service 'kolide_launcher_service' that will execute the 'launcher.exe'
                with the arguments that has been provided

            Example 02:
            - > wrapper.exe kolide_launcher_service remove
				This will stop and remove 'kolide_launcher_service' service
