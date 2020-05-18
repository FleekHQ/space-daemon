# API Structure

POC structure for API follows clean architecture pattern where handlers and controllers 
know about the web framework but business logic and internals are on domain and services layer

`/app` Package for app bootstrap and initialization. This package knows about the web framework
`/controllers` Package for web framework handler call and mapping to prepare request and responses from Service Layer
`/domain` Domain Models and core  business logic for API. Does not import any web framework dependency unless 
its through an interface
`/services` Business Logic code that handles request from controller layer and interacts with domain to 
serve back a response. Only dependency should be domain and data layers or interact with external dependencies
with interfaces as much as possible;
`/logger` Setup logger dependency through a package interface that can be used in the app.

Other Potential directories to consider adding as needed:
`/util` For utils or helpers that don't make sense to be inside other packages
`/data` or `/datasource` For data access layer if needed
