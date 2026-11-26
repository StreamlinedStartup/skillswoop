# swoop

swoop finds skills and plugins and installs them into the agents a person codes with, per project or globally.

## Language

### Things you install

**Skill**:
A folder of instructions an **Agent** can load.

**Plugin**:
A bundle for an **Agent** that can include **Skills**, hooks, MCP servers, and commands.

**Agent**:
A coding tool that loads **Skills** and **Plugins**, such as Claude Code or Codex.
_Avoid_: Harness, assistant, tool

**Default agents**:
The **Agents** an install goes to unless the person picks others.

### Where things come from

**Source**:
A place **Skills** come from: a GitHub repo, a git URL, a local folder, or the **Library**.
_Avoid_: Repo (when meaning a remembered skill location)

**Marketplace**:
A place **Plugins** come from. A repo that holds both skills and plugins is remembered once as a **Source** and once as a **Marketplace**.

**Library**:
The one built-in **Source** that holds **Skills** added from the person's own machine or moved there by **Tidy**. It cannot be removed.

**Alias**:
A display name a person gives a **Source**. It does not change the **Source** or its **Skills**.
_Avoid_: Rename (for this meaning)

**Star**:
One **Skill** from one **Source**, marked so it can be installed again quickly.
_Avoid_: Favorite, bookmark

### Where things go

**Project**:
A folder that **Skills** have been installed into. swoop remembers every **Project**.

**Install destination**:
Where an install goes: **This project** (the current folder) or **Globally** (every project for the chosen **Agents**).
_Avoid_: Scope

### Actions

**Tidy**:
Moving global **Skills** out of the **Agents**' global folders into the **Library**.
_Avoid_: Migrate

**Rename**:
Changing a **Skill**'s real name, the name **Agents** see.
_Avoid_: Set alias

**Name clash**:
Two different **Skills** with the same name meeting in the **Library**.

**Update this folder**:
Refreshing the **Skills** installed in the current **Project**.

**Update global**:
Refreshing the **Skills** installed **Globally**.

**Update everything**:
Refreshing the **Skills** in every **Project** and the global ones.
