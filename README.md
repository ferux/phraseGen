# phraseGen

Phrase Gen app uses markov chains to generate new phrases according to it's own
matrix of words.  

## Settings

It's possible to setup application in a two ways:  

1. Set environment variables manually;
1. Set these variables in .env file (but you need to rename .env.example to .env);

Also you should specify location to the file which contains plain text / parsed text
in JSON format with the following model:  

```JSON
[
        {
                "number": 0,
                "date": "anydate",
                "text": "text"
        }
]
```

It will skip all fields except field "Text", which will be parsed and applied to the
chain.  

To specify the file you have a few options:

1. Setting env variable "GO_FILE";
1. Setting the flag "file" when launching application;
