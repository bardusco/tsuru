Tsuru Overview
==============

Tsuru is an OpenSource PaaS. If you don't know what a PaaS is and what it does, see `wikipidia's description <http://en.wikipedia.org/wiki/PaaS>`_.

It follows the principles described in the `The Twelve-Factor App <http://www.12factor.net/>`_ methodology.

Immediate Deploy
----------------

Deploying an app is simple and easy. No special tools needed, just a plain git push. Deployment is instant, whether your app is big or small.

Tsuru uses git as the means of deploying an application. You don't need master git in order to deploy an app to Tsuru, although you will need to know
the very basic workflow, add/commit/push and remote managing. Git allows really fast deploys, and Tsuru makes the best possible use of it by not cloning
the whole repository history of your application, there's no need to have that information in the application webserver.

Tsuru will also take care of all the applications dependencies when you deploy. You can specify Operational System and language specific dependencies.
For example, if you have a Python application, tsuru will search for the requirements.txt file, but first it will search for OS dependencies. in Ubuntu, for instance,
Tsuru will search for a file called requirements.apt, after the installation of the packages listed there, it'll install the language dependencies, in this example
running `pip instal` passing the requirements.txt file.

Tsuru also has `deploy hooks <https://tsuru.readthedocs.org/en/latest/apps/client/usage.html#adding-hooks>`_ that can trigger commands before and after the restart of your application.

Continuos Deploy
----------------

Easily create testing, staging, and production versions of your app and deploy to and between them instantly.

Add-on Resources
----------------

Instantly provision and integrate third party services with one command. Tsuru provides the basic services your application will need, like
Databases, Search, Caching, Storage and FrontEnd; you can get all of that in a fashionable and really easy way using Tsuru's command line.

Per-Deploy Config Variables
---------------------------

Configuration for an application should be stored in environment variables - and we know that. Tsuru lets you define your environment variables using the command line,
so you can have the configuration flexibility your application need.

Additionaly, when you bind a service with your application, Tsuru gives the service the hability to inject environment variables in your application environment.
For instance, if you use our Mysql service, it will inject variables for you to create a connection with your application database.

Custom Services
---------------

Tsuru already has services for you to use, but you don't need to use them at all if you don't want to. If you already have, let's say,
a Mysql server running on your infrastructure, all you need to do in order to use it is simply configure environment variables and use
them in your application config.

You can also create your own services and make them available for others to use it on Tsuru. It's so easy to do so that you'll want to sell
your own services. Tsuru talks with services using a well defined API, all you have to do is implement four endpoints that knows how to
provision your service to an application (like creating VMs, liberating access, etc), and register your service in Tsuru with a really simple
`yaml manifest <https://github.com/globocom/varnishapi/blob/master/manifest.yaml>`_.

Logging and Visibility
----------------------

Full visibility into your app's operations with real-time logging, process status inspection, and an audit trail of all releases.
Tsuru will log everything related to your application, and you can check it out simply running `tsuru log` command. You can filter logs, for example,
if you don't want to see the logs of developers activity (e.g.: a deploy action), you can specify the source as `app` and you'll get only the application webserver logs.

Process Management
------------------

Tsuru manages the application process for your application so you don't have to worry about it. But it does not know to start it.
You'll have to teach Tsuru how to start your application using a Procfile. Tsuru reads the Procfile and uses Circus_ to start and manage the running process.
You can even enable a web console for Circus to manage your application process and to watch CPU and memory usage in real-time through a web interface.

Tsuru also allows you to easily restart your application process by the command line. Although Tsuru will do all the hard work of managing and fixing eventual
problems with your process, you might need to restart your application manually, so we give you an easy way to do it.

.. _Circus: http://circus.readthedocs.org

Control Surfaces
----------------

Tsuru offers two main ways of interacting with it: the CLI interface, which is awesome for day-to-day usage; and the web interface, which is in alpha,
but is also a great tool to manage, check logs and monitor applications and services resources.
Of course Tsuru have a solid API, so if you want to make your own tools, it's possible!

Scaling
-------

The Juju_ provisioner allows you to easily add and remove units, allowing one to scale an application painlessly. It will take care of the
deploy of the application code to the new units and with the bind with the used services, there's nothing required to the developer to do in order
to scale a application, just add a new unit and Tsuru will do the trick.

You may also want to scale using the FrontEnd as a Service, powered by `Varnish <https://www.varnish-cache.org/>`_. One single application might have a whole farm of Varnish VMs in
front of it receiving all the traffic.

Collaboration
-------------

Manage sharing and deployment of your application. Tsuru has teams concept, you can create your team and add your teammates.
One can be on various teams and control which applications the teams has permissions.

Easy Server Deployment
----------------------

Tsuru itself is really easy to deploy, you can get it done by following `these simple steps <http://docs.tsuru.io/en/latest/build.html>`_.

Distributed and Extensible
--------------------------

Tsuru server is easily extensible and customizable. By default, it will deploy applications using the Juju_ provisioner, but you can easily implement your own
provisioner and use whenever backend you wish.

When you extend Tsuru, you are praticaly building a new PaaS, in terms of behavior when provisioning and , reusing Tsuru's structure. You change the whole Tsuru workflow by implementing a new provider.
Tsuru allows it with the power of Golang, it uses interfaces, so you can just create your own provisioner respecting Tsuru's interface and plug in it, changing your PaaS
behavior.

.. _Juju: https://juju.ubuntu.com/

Dev/Ops Perspective
-------------------

Tsuru's components are distributed, it is formed by various pieces of software, each one made to be easily deployed and maintained.

Application Developer Perspective
---------------------------------

We aim to make developers life easier.
