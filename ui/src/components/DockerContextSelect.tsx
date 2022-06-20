import React, { useEffect } from "react";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import Select, { SelectChangeEvent } from "@mui/material/Select";
import { createDockerDesktopClient } from "@docker/extension-api-client";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export interface DockerContext {
  Current: boolean;
  Description: string;
  DockerEndpoint: string;
  KubernetesEndpoint: string;
  ContextType: string;
  Name: string;
  StackOrchestrator: string;
}

export default function DockerContextSelect({ ...props }) {
  console.log("DockerContextSelect component rendered.");
  const ddClient = useDockerDesktopClient();

  const [dockerContexts, setDockerContexts] = React.useState<DockerContext[]>(
    []
  );

  const dockerContextName = props.dockerContextName;

  const handleChange = (event: SelectChangeEvent) => {
    props.setDockerContextName(event.target.value as string);
  };

  useEffect(() => {
    const listDockerContexts = async () => {
      const output = await ddClient.docker.cli.exec("context", [
        "ls",
        "--format",
        "json",
      ]);

      const dockerContexts = output.parseJsonObject() as DockerContext[];
      setDockerContexts(dockerContexts);
    };

    listDockerContexts();
  }, []);

  return (
    <Grid container mt={2}>
      <Grid item>
        <FormControl fullWidth>
          <InputLabel id="docker-context-select-label">
            Docker Context
          </InputLabel>
          <Select
            labelId="docker-context-select-label"
            id="docker-context-select"
            value={dockerContextName}
            label="Docker Context Name"
            onChange={handleChange}
          >
            {dockerContexts.map((ctx) => {
              return (
                <MenuItem value={ctx.Name}>
                  {ctx.Name} ({ctx.DockerEndpoint})
                </MenuItem>
              );
            })}
          </Select>
        </FormControl>
      </Grid>
    </Grid>
  );
}
