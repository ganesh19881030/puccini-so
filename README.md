CCI ONAP Service Orchestration
=============================================================

In light of our current experience with ONAP, we have come to the conclusion that there must be a better way to
orchestrate network services.  It is too complex, disjointed, and brittle to be of any practical use in the industry.
We are embarking on a monumental task of simplifying its architecture

Our focus is on the following main areas:

1. Orchestration:
-----------------

It is our belief that ONAP is overly complex and has too many components that perform similar orchestration functions. In ONAP, there is the Service Orchestrator delegating to what we can think of as domain-specific orchestrators - SDNC for virtual networks, VNFM for virtual network functions, APPC for applications, etc.  We believe, we can replace the main orchestrator along with the domain-specific orchestrators with a common orchestration engine that is based on the declarative TOSCA language for orchestration logic. The orchestration engine will orchestrate based on the TOSCA template for that domain.

2. Workflows & Policies
-----------------------
Since TOSCA specification's support for workflows and policies is more than adequate for what is needed, we can significantly reduce the complexity of the current ONAP implementation which depends on complex and full-blown workflow engines like Camunda, and Drools based policy engines to fulfill this requirement.

3. Active and available Inventory
---------------------------------

We are building a schema for persisting TOSCA based object models in a Graph DB. Our goal here is to store the parsed/compiled content of a TOSCA model as expressed in a "clout" file from Puccini compiler, into the graph db. The "schema" will be formulated in terms of the TOSCA entities, which would avoid redefining/changing the schema everytime we need to add a new resource.  As long the resource is defined in terms of the TOSCA entities we should be covered. 

Although initially, we are building this database from the perspective of orchestration alone, eventually we plan to replace the active and available inventory of ONAP with it.

We think if we successfully execute these ideas, we can reduce the complexity of ONAP by several orders of magnitude, thus making it easier to adopt in the industry.

As you can tell the key to the success of the project is TOSCA and its available tooling.  We have found open source project PUCCINI to be the best resource for TOSCA parsing, compiling, and orchestration. Our plan is to leverage Puccini as much as possible to implement the TOSCA based service orchestrator.
