# distributed-dns
A repository containing our learnings and implementations for the project "Distributed DNS in the Cloud" under IEEE-NITK

[AWS Architecture Explanation](https://drive.google.com/file/d/1z0d43xg4ED1ooZYSyzBbUR22-izMiX0v/view?usp=sharing)  
  
[Writing a record into one set of nameservers](https://drive.google.com/file/d/1IKcLwHWtbH5pK-MQGfGREsp28uGwRIyF/view?usp=sharing)  
We perform the above process for each step of resolution of the record (eg., here to the root nameservers, we write the address of the Global Accelerator responsible for routing the requests to the "com" nameservers, to the com nameservers, we write the address of the Global Accelerator responsible for routing the requests to the "example.com" nameservers, to the example.com nameservers, we write the final A record).
  
[Resolving a name](https://drive.google.com/file/d/1OSD5CHe4Z_iHownfl_0Kr6wKq9Gk9x6L/view?usp=sharing)
