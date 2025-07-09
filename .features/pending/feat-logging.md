Component: General
Issues: 11120
Description: This migrates most of the logging off logrus and onto a custom logger.
Author: [Isitha Subasinghe](https://github.com/isubasinghe)

Currently it is quite hard to identify log lines with it's corresponding workflow.
This change propagates a context object down the call hierarchy containing an 
annotated logging object. This allows context aware logging from deep within the
codebase.
