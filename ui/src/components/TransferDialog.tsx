import React, { useContext, useEffect } from "react";
import {
  Autocomplete,
  Button,
  TextField,
  Typography,
  Backdrop,
  CircularProgress,
  Alert,
  FormControl,
  FormLabel,
  InputAdornment,
  Stack,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { isError } from "../common/isError";
import { VolumeOrInput } from "./VolumeOrInput";
import { useNotificationContext } from "../NotificationContext";

const ddClient = createDockerDesktopClient();

export default function TransferDialog({ ...props }) {
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();

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
  // eslint-disable-next-line react-hooks/exhaustive-deps
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
        sendNotification(listVolumesOutput.stderr);
        return;
      }
      return listVolumesOutput.lines();
    } catch (error) {
      sendNotification(
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
        `"cd /from ; tar -czf - . " | ssh ${destHost} "docker run --rm -i -v "${volumeName}":/to alpine ash -c 'cd /to ; tar -xpvzf - '"`,
      ]);
      if (isError(transferredOutput.stderr)) {
        sendNotification(transferredOutput.stderr);
        return;
      }

      sendNotification(
        `Volume ${context.store.volume.volumeName} transferred to destination volume ${volumeName} in host ${destHost}`
      );
    } catch (error) {
      sendNotification(
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
        <Alert
          sx={(theme) => ({ marginBottom: theme.spacing(2) })}
          severity="warning"
        >
          Any existing data inside the destination volume will be replaced.
        </Alert>
        <FormControl>
          <FormLabel id="from-label">
            <Typography variant="h3" my={1}>
              From:
            </Typography>
          </FormLabel>
          <VolumeOrInput />
        </FormControl>
        <FormControl sx={{ width: "100%" }}>
          <FormLabel id="to-label">
            <Typography variant="h3" mt={3} mb={1}>
              To:
            </Typography>
          </FormLabel>
          <Stack mt={1} spacing={3}>
            <TextField
              fullWidth
              autoFocus
              id="dest-host"
              label="Destination host"
              helperText="SSH must be enabled and configured between the source and destination Docker hosts. Check you have the remote host SSH public key in your known_hosts file."
              placeholder={"user@192.168.1.50"}
              spellCheck={false}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">ssh://</InputAdornment>
                ),
              }}
              value={destHost}
              onChange={(event) => setDestHost(event.target.value)}
            />
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
          </Stack>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button variant="outlined" onClick={props.onClose}>
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={transferVolume}
          disabled={volumeName === ""}
        >
          Transfer
        </Button>
      </DialogActions>
    </Dialog>
  );
}
