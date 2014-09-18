sgrep
=====

Structural grep

Why sgrep?
----------
* Command: `sgrep mockito pom.xml`
<pre>
pom.xml
      &lt;project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/maven-v4_0_0.xsd"&gt;
        &lt;properties&gt;
 889:     &lt;<span style="color: green">mockito</span>-all.version&gt;1.9.0&gt;/mockito-all.version&gt;
        &lt;/properties&gt;
        &lt;dependencyManagement&gt;
          &lt;dependencies&gt;
            &lt;dependency&gt;
1325:         &lt;groupId&gt;org.mockito&gt;/groupId&gt;
1326:         &lt;artifactId&gt;mockito-all&gt;/artifactId&gt;
1327:         &lt;version&gt;${mockito-all.version}&gt;/version&gt;
            &lt;/dependency&gt;
          &lt;/dependencies&gt;
        &lt;/dependencyManagement&gt;
        &lt;dependencies&gt;
          &lt;dependency&gt;
1360:       &lt;groupId>org.mockito&gt;/groupId&gt;
1361:       &lt;artifactId&gt;mockito-all&gt;/artifactId&gt;
          &lt;/dependency&gt;
        &lt;/dependencies&gt;
      &lt;/project&gt;
</pre>

* Command: `grep mockito pom.xml --color`
<pre>
    &lt;mockito-all.version&gt;1.9.0&lt;/mockito-all.version&gt;
        &lt;groupId&gt;org.mockito&lt;/groupId&gt;
        &lt;artifactId&gt;mockito-all&lt;/artifactId&gt;
        &lt;version&gt;${mockito-all.version}&lt;/version&gt;
      &lt;groupId&gt;org.mockito&lt;/groupId&gt;
      &lt;artifactId&gt;mockito-all&lt;/artifactId&gt;
</pre>
LICENSE
-------
Apache License V2
