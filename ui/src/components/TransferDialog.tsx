import React, { useContext, useEffect } from "react";
import {
  Autocomplete,
  Button,
  TextField,
  Typography,
  Grid,
  Backdrop,
  CircularProgress,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { isError } from "../common/isError";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function TransferDialog({ ...props }) {
  const ddClient = useDockerDesktopClient();
  const context = useContext(MyContext);

  const [volumeName, setVolumeName] = React.useState<string>("");
  const [destHost, setDestHost] = React.useState<string>("");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const [autocompleteOpen, setAutocompleteOpen] = React.useState(false);
  const [options, setOptions] = React.useState<readonly string[]>([]);
  const autocompleteLoading = autocompleteOpen && options.length === 0;

  useEffect(() => {
    let active = true;

    if (!autocompleteLoading) {
      return undefined;
    }

    (async () => {
      const volumes = await listVolumesForDockerHost();

      if (active) {
        setOptions([...volumes]);
      }
    })();

    return () => {
      active = false;
    };
  }, [autocompleteLoading]);

  useEffect(() => {
    if (!autocompleteOpen) {
      setOptions([]);
    }
  }, [autocompleteOpen]);

  const listVolumesForDockerHost = async () => {
    try {
      // docker -H ssh://192.168.1.50 volume ls --format="{{ .Name }}"
      const listVolumesOutput = await ddClient.docker.cli.exec("-H", [
        `ssh://${destHost}`,
        "volume",
        "ls",
        `--format="{{ .Name }}"`,
      ]);

      if (listVolumesOutput.stderr !== "") {
        ddClient.desktopUI.toast.error(listVolumesOutput.stderr);
        return;
      }
      return listVolumesOutput.lines();
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Unable to list volumes for docker host ${destHost}: ${error.stderr} Exit code: ${error.code}`
      );
      return [];
    }
  };

  const transferVolume = async () => {
    setActionInProgress(true);

    try {
      console.log(
        `Transferring data from source volume ${context.store.volume.volumeName} to destination volume ${volumeName} in host ${destHost}...`
      );

      // docker run --rm \
      //      -v dockprom_prometheus_data:/from alpine ash -c \
      //      "cd /from ; tar -czf - . " | \
      //      ssh 192.168.1.50 \
      //      "docker run --rm -i -v \"rpi-vol-2\":/to alpine ash -c 'cd /to ; tar -xpvzf - '"

      const transferredOutput = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${context.store.volume.volumeName}:/from`,
        "alpine",
        "ash",
        "-c",
        `"cd /from ; tar -czf - . \" | ssh ${destHost} \"docker run --rm -i -v \"${volumeName}\":/to alpine ash -c 'cd /to ; tar -xpvzf - '"`,
      ]);
      if (isError(transferredOutput.stderr)) {
        ddClient.desktopUI.toast.error(transferredOutput.stderr);
        return;
      }

      ddClient.desktopUI.toast.success(
        `Volume ${context.store.volume.volumeName} transferred to destination volume ${volumeName} in host ${destHost}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to clone volume ${context.store.volume.volumeName} to destinaton volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Transfer a volume between Docker hosts</DialogTitle>
      <DialogContent>
        <Backdrop
          sx={{
            backgroundColor: "rgba(245,244,244,0.4)",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
          open={actionInProgress}
        >
          <CircularProgress color="info" />
        </Backdrop>
        <DialogContentText>
          SSH must be enabled and configured between the source and destination
          Docker hosts.
        </DialogContentText>

        <Grid container direction="column" spacing={2} mt={2}>
          <Grid
            container
            direction="row"
            alignItems="flex-end"
            textAlign="center"
            justifyContent="center"
            spacing={1}
          >
            <Grid item>
              <Button variant="outlined" disabled>
                ssh://
              </Button>
            </Grid>
            <Grid item>
              <TextField
                required
                autoFocus
                margin="dense"
                id="dest-host"
                label="Destination host"
                fullWidth
                variant="standard"
                placeholder={"192.168.1.50"}
                spellCheck={false}
                onChange={(e) => {
                  setDestHost(e.target.value);
                }}
              />
            </Grid>
          </Grid>
          <Grid item>
            <Autocomplete
              id="autocomplete-destination-volume"
              open={autocompleteOpen}
              onOpen={() => {
                setAutocompleteOpen(true);
              }}
              onClose={() => {
                setAutocompleteOpen(false);
              }}
              isOptionEqualToValue={(option, value) => option === value}
              getOptionLabel={(option) => option}
              options={options}
              loading={autocompleteLoading}
              disabled={destHost === ""}
              inputValue={volumeName}
              onInputChange={(event, newInputValue) => {
                setVolumeName(newInputValue);
              }}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Destination volume"
                  InputProps={{
                    ...params.InputProps,
                    endAdornment: (
                      <React.Fragment>
                        {autocompleteLoading ? (
                          <CircularProgress color="inherit" size={20} />
                        ) : null}
                        {params.InputProps.endAdornment}
                      </React.Fragment>
                    ),
                  }}
                />
              )}
            />
          </Grid>
          {volumeName !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                The volume will be transferred to an existing volume named{" "}
                {volumeName} in host {destHost}.
              </Typography>
              <Typography variant="body1" color="text.secondary">
                ⚠️ This will replace all the existing data inside the existing
                volume.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Cancel</Button>
        <Button onClick={transferVolume} disabled={volumeName === ""}>
          Transfer
        </Button>
      </DialogActions>
    </Dialog>
  );
}
